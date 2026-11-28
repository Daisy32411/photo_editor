import base64
import io
import os
import time
import uuid

import pytest
import requests
from PIL import Image


BASE_URL = os.getenv("API_URL", "http://localhost:8001")


def create_test_image():
    image = Image.new("RGB", (4, 4), (10, 20, 30))

    for y in range(2, 4):
        for x in range(4):
            image.putpixel((x, y), (200, 100, 50))

    buffer = io.BytesIO()
    image.save(buffer, format="PNG")

    return base64.b64encode(buffer.getvalue()).decode("ascii")


def create_user_headers():
    username = f"test_{uuid.uuid4().hex[:12]}"
    password = "test123456"

    response = requests.post(
        f"{BASE_URL}/register",
        json={
            "username": username,
            "password": password,
        },
        timeout=10,
    )

    assert response.status_code == 201, response.text

    response = requests.post(
        f"{BASE_URL}/login",
        json={
            "username": username,
            "password": password,
        },
        timeout=10,
    )

    assert response.status_code == 200, response.text

    token = response.json()["token"]

    assert len(token) == 64

    return {
        "Authorization": f"Bearer {token}",
    }


@pytest.fixture(scope="module")
def auth_headers():
    return create_user_headers()


@pytest.fixture
def other_auth_headers():
    return create_user_headers()


@pytest.fixture(scope="module")
def processor_headers():
    token = os.getenv("PROCESSOR_TOKEN")

    if not token:
        pytest.fail("PROCESSOR_TOKEN environment variable is not set")

    return {
        "X-Processor-Token": token,
    }


def create_task(headers, filter_name="Negative", parameters=None, image=None):
    filter_data = {
        "name": filter_name,
    }

    if parameters is not None:
        filter_data["parameters"] = parameters

    payload = {
        "image": image if image is not None else create_test_image(),
        "filter": filter_data,
    }

    response = requests.post(
        f"{BASE_URL}/task",
        headers=headers,
        json=payload,
        timeout=20,
    )

    assert response.status_code == 201, response.text

    task_id = response.json()["task_id"]

    assert uuid.UUID(task_id)

    return task_id


def wait_for_task(task_id, headers, timeout=60, expected="ready"):
    deadline = time.monotonic() + timeout

    while time.monotonic() < deadline:
        response = requests.get(
            f"{BASE_URL}/status/{task_id}",
            headers=headers,
            timeout=5,
        )

        assert response.status_code == 200, response.text

        status = response.json()["status"]

        if status == expected:
            return status

        assert status == "in_progress", (
            f"Unexpected status: {status}, expected: {expected}"
        )

        time.sleep(0.5)

    pytest.fail(f"Task {task_id} timed out after {timeout} seconds")


@pytest.mark.parametrize(
    "filter_name,parameters",
    [
        ("Negative", None),
        ("Blur", {"sigma": 2.5}),
        ("Sharpen", {"sigma": 1.5}),
        ("FlipX", None),
    ],
)
def test_image_processing(auth_headers, filter_name, parameters):
    task_id = create_task(
        auth_headers,
        filter_name,
        parameters,
    )

    assert wait_for_task(task_id, auth_headers) == "ready"

    response = requests.get(
        f"{BASE_URL}/result/{task_id}",
        headers=auth_headers,
        timeout=30,
    )

    assert response.status_code == 200, response.text
    assert response.headers["Content-Type"] == "image/png"
    assert response.content.startswith(b"\x89PNG\r\n\x1a\n")

    image = Image.open(io.BytesIO(response.content))
    image.load()

    assert image.format == "PNG"
    assert image.size == (4, 4)

    rgb_image = image.convert("RGB")

    if filter_name == "Negative":
        assert rgb_image.getpixel((0, 0)) == (245, 235, 225)
        assert rgb_image.getpixel((0, 3)) == (55, 155, 205)

    if filter_name == "FlipX":
        assert rgb_image.getpixel((0, 0)) == (200, 100, 50)
        assert rgb_image.getpixel((0, 3)) == (10, 20, 30)


@pytest.mark.parametrize("endpoint", ["status", "result"])
def test_get_task_unauthorized(endpoint):
    response = requests.get(
        f"{BASE_URL}/{endpoint}/{uuid.uuid4()}",
        timeout=10,
    )

    assert response.status_code == 401, response.text


def test_create_task_unauthorized():
    response = requests.post(
        f"{BASE_URL}/task",
        json={
            "image": create_test_image(),
            "filter": {
                "name": "Negative",
            },
        },
        timeout=10,
    )

    assert response.status_code == 401, response.text


@pytest.mark.parametrize("endpoint", ["status", "result"])
def test_task_not_found(auth_headers, endpoint):
    response = requests.get(
        f"{BASE_URL}/{endpoint}/{uuid.uuid4()}",
        headers=auth_headers,
        timeout=10,
    )

    assert response.status_code == 404, response.text


@pytest.mark.parametrize("endpoint", ["status", "result"])
def test_invalid_task_id(auth_headers, endpoint):
    response = requests.get(
        f"{BASE_URL}/{endpoint}/invalid-uuid",
        headers=auth_headers,
        timeout=10,
    )

    assert response.status_code == 400, response.text


def test_task_ownership(auth_headers, other_auth_headers):
    assert auth_headers != other_auth_headers

    task_id = create_task(auth_headers)

    assert wait_for_task(task_id, auth_headers) == "ready"

    owner_status = requests.get(
        f"{BASE_URL}/status/{task_id}",
        headers=auth_headers,
        timeout=10,
    )

    assert owner_status.status_code == 200
    assert owner_status.json()["status"] == "ready"

    owner_result = requests.get(
        f"{BASE_URL}/result/{task_id}",
        headers=auth_headers,
        timeout=10,
    )

    assert owner_result.status_code == 200
    assert owner_result.headers["Content-Type"] == "image/png"

    other_status = requests.get(
        f"{BASE_URL}/status/{task_id}",
        headers=other_auth_headers,
        timeout=10,
    )

    assert other_status.status_code == 404

    other_result = requests.get(
        f"{BASE_URL}/result/{task_id}",
        headers=other_auth_headers,
        timeout=10,
    )

    assert other_result.status_code == 404


@pytest.mark.parametrize(
    "auth_type",
    ["missing", "invalid", "user"],
)
def test_commit_unauthorized(auth_headers, auth_type):
    task_id = create_task(auth_headers)

    assert wait_for_task(task_id, auth_headers) == "ready"

    original = requests.get(
        f"{BASE_URL}/result/{task_id}",
        headers=auth_headers,
        timeout=10,
    )

    assert original.status_code == 200

    headers = {}

    if auth_type == "invalid":
        headers = {
            "X-Processor-Token": "wrong-token",
        }

    if auth_type == "user":
        headers = auth_headers

    response = requests.post(
        f"{BASE_URL}/commit",
        headers=headers,
        json={
            "task_id": task_id,
            "status": "failed",
            "result": "",
        },
        timeout=10,
    )

    assert response.status_code == 401, response.text

    current = requests.get(
        f"{BASE_URL}/result/{task_id}",
        headers=auth_headers,
        timeout=10,
    )

    assert current.status_code == 200
    assert current.content == original.content


def test_commit_not_found(processor_headers):
    response = requests.post(
        f"{BASE_URL}/commit",
        headers=processor_headers,
        json={
            "task_id": str(uuid.uuid4()),
            "status": "failed",
            "result": "",
        },
        timeout=10,
    )

    assert response.status_code == 404, response.text


def test_commit_invalid_uuid(processor_headers):
    response = requests.post(
        f"{BASE_URL}/commit",
        headers=processor_headers,
        json={
            "task_id": "invalid-uuid",
            "status": "failed",
            "result": "",
        },
        timeout=10,
    )

    assert response.status_code == 400, response.text


def test_commit_idempotency(auth_headers, processor_headers):
    task_id = create_task(auth_headers)

    assert wait_for_task(task_id, auth_headers) == "ready"

    original = requests.get(
        f"{BASE_URL}/result/{task_id}",
        headers=auth_headers,
        timeout=10,
    )

    assert original.status_code == 200

    original_result = base64.b64encode(
        original.content
    ).decode("ascii")

    response = requests.post(
        f"{BASE_URL}/commit",
        headers=processor_headers,
        json={
            "task_id": task_id,
            "status": "ready",
            "result": original_result,
        },
        timeout=10,
    )

    assert response.status_code == 204, response.text

    response = requests.post(
        f"{BASE_URL}/commit",
        headers=processor_headers,
        json={
            "task_id": task_id,
            "status": "ready",
            "result": create_test_image(),
        },
        timeout=10,
    )

    assert response.status_code == 409, response.text

    response = requests.post(
        f"{BASE_URL}/commit",
        headers=processor_headers,
        json={
            "task_id": task_id,
            "status": "failed",
            "result": "",
        },
        timeout=10,
    )

    assert response.status_code == 409, response.text

    current = requests.get(
        f"{BASE_URL}/result/{task_id}",
        headers=auth_headers,
        timeout=10,
    )

    assert current.status_code == 200
    assert current.content == original.content


@pytest.mark.parametrize(
    "body",
    [
        '{"image":',
        '{"image":"abc","filter":{"name":"Negative"}} {}',
    ],
)
def test_invalid_json(auth_headers, body):
    response = requests.post(
        f"{BASE_URL}/task",
        headers={
            **auth_headers,
            "Content-Type": "application/json",
        },
        data=body,
        timeout=10,
    )

    assert response.status_code == 400, response.text


@pytest.mark.parametrize(
    "invalid_image",
    [
        "!!!",
        base64.b64encode(b"not a PNG").decode("ascii"),
    ],
)
def test_invalid_image(auth_headers, invalid_image):
    task_id = create_task(
        auth_headers,
        image=invalid_image,
    )

    assert wait_for_task(
        task_id,
        auth_headers,
        expected="failed",
    ) == "failed"

    response = requests.get(
        f"{BASE_URL}/result/{task_id}",
        headers=auth_headers,
        timeout=10,
    )

    assert response.status_code == 409, response.text