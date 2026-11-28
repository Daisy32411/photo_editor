import json
import os
import subprocess
import time
import uuid
from contextlib import contextmanager

import pika
import requests

from test import (
    BASE_URL,
    create_user_headers,
    create_task,
    wait_for_task,
)


RABBITMQ_URL = os.getenv(
    "RABBITMQ_TEST_URL",
    "amqp://photo:photo@localhost:5672/",
)

TASK_QUEUE = "image_tasks"
DLQ = "image_tasks_dlq"
DLX = "image_tasks_dlx"


def compose(*args):
    result = subprocess.run(
        ["docker", "compose", *args],
        capture_output=True,
        text=True,
        timeout=180,
    )

    assert result.returncode == 0, result.stderr

    return result.stdout.strip()


def wait_until(predicate, timeout=60):
    deadline = time.monotonic() + timeout

    while time.monotonic() < deadline:
        result = predicate()

        if result:
            return result

        time.sleep(0.5)

    raise AssertionError("Condition was not met before timeout")


def api_ready():
    try:
        response = requests.get(
            f"{BASE_URL}/metrics",
            timeout=2,
        )

        return response.status_code == 200

    except requests.RequestException:
        return False


@contextmanager
def rabbit_channel():
    parameters = pika.URLParameters(RABBITMQ_URL)

    parameters.connection_attempts = 1
    parameters.socket_timeout = 5
    parameters.stack_timeout = 10

    connection = pika.BlockingConnection(parameters)

    try:
        yield connection.channel()
    finally:
        if connection.is_open:
            connection.close()


def processor_ready():
    try:
        with rabbit_channel() as channel:
            queue = channel.queue_declare(
                queue=TASK_QUEUE,
                passive=True,
            )

            return queue.method.consumer_count > 0

    except (pika.exceptions.AMQPError, OSError):
        return False


def server_identity():
    container_id = compose("ps", "-q", "server")

    assert container_id

    result = subprocess.run(
        [
            "docker",
            "inspect",
            "-f",
            "{{.State.StartedAt}}",
            container_id,
        ],
        capture_output=True,
        text=True,
        check=True,
        timeout=10,
    )

    return container_id, result.stdout.strip()


def publish_message(channel, body, message_id):
    channel.confirm_delivery()

    channel.basic_publish(
        exchange="",
        routing_key=TASK_QUEUE,
        body=body,
        mandatory=True,
        properties=pika.BasicProperties(
            content_type="application/json",
            delivery_mode=2,
            message_id=message_id,
        ),
    )


def dlq_message_count(channel):
    queue = channel.queue_declare(
        queue=DLQ,
        passive=True,
    )

    return queue.method.message_count


def create_probe_queue(channel):
    queue = channel.queue_declare(
        queue="",
        exclusive=True,
        auto_delete=True,
    )

    queue_name = queue.method.queue

    channel.queue_bind(
        queue=queue_name,
        exchange=DLX,
        routing_key=DLQ,
    )

    return queue_name


def wait_for_dead_letter(channel, queue_name, message_id):
    def receive():
        method, properties, body = channel.basic_get(
            queue=queue_name,
            auto_ack=True,
        )

        if method is None:
            return None

        if properties.message_id != message_id:
            return None

        return properties

    return wait_until(receive, timeout=60)


def assert_death_reason(properties, expected_reason):
    headers = properties.headers or {}

    deaths = headers.get("x-death", [])

    assert any(
        death.get("queue") == TASK_QUEUE
        and death.get("reason") == expected_reason
        for death in deaths
    ), f"Unexpected x-death headers: {deaths}"


def test_rabbitmq_reconnect():
    wait_until(api_ready)
    wait_until(processor_ready)

    headers = create_user_headers()

    first_task = create_task(headers)

    assert wait_for_task(first_task, headers) == "ready"

    original_server = server_identity()

    compose("restart", "rabbitmq")

    compose(
        "up",
        "-d",
        "--wait",
        "--wait-timeout",
        "120",
        "rabbitmq",
    )

    wait_until(processor_ready)
    wait_until(api_ready)

    second_task = create_task(headers)

    assert wait_for_task(second_task, headers) == "ready"

    assert server_identity() == original_server, (
        "Server was restarted instead of reconnecting"
    )


def test_invalid_message_dlq():
    wait_until(processor_ready)

    message_id = str(uuid.uuid4())

    with rabbit_channel() as channel:
        probe_queue = create_probe_queue(channel)

        original_count = dlq_message_count(channel)

        publish_message(
            channel,
            b"invalid-json",
            message_id,
        )

        properties = wait_for_dead_letter(
            channel,
            probe_queue,
            message_id,
        )

        assert_death_reason(properties, "rejected")

        wait_until(
            lambda: dlq_message_count(channel) > original_count
        )


def test_retry_limit():
    wait_until(api_ready)
    wait_until(processor_ready)

    message_id = str(uuid.uuid4())

    message = {
        "task_id": str(uuid.uuid4()),
        "image": "!!!",
        "filter": {
            "name": "Negative",
        },
    }

    try:
        compose("stop", "server")

        with rabbit_channel() as channel:
            probe_queue = create_probe_queue(channel)

            original_count = dlq_message_count(channel)

            publish_message(
                channel,
                json.dumps(message).encode("utf-8"),
                message_id,
            )

            properties = wait_for_dead_letter(
                channel,
                probe_queue,
                message_id,
            )

            assert_death_reason(
                properties,
                "delivery_limit",
            )

            wait_until(
                lambda: dlq_message_count(channel) > original_count
            )

    finally:
        compose("start", "server")

        wait_until(api_ready)