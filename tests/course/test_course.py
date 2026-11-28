import base64
import io
import os
import time
import uuid
from pathlib import Path

import pytest
import requests
from PIL import Image


BASE_URL = os.getenv("API_URL", "http://127.0.0.1:8001")
IMAGE_PATH = Path(__file__).parent / "static" / "sigma.png"


@pytest.fixture
def user_data():
    return {
        "username": f"user_{uuid.uuid4().hex[:12]}",
        "password": "password228",
    }


@pytest.fixture
def registered_user(user_data):
    response = requests.post(
        f"{BASE_URL}/register",
        json=user_data,
        timeout=10,
    )

    assert response.status_code == 201, response.text

    return user_data


@pytest.fixture
def auth_token(registered_user):
    response = requests.post(
        f"{BASE_URL}/login",
        json=registered_user,
        timeout=10,
    )

    assert response.status_code == 200, response.text

    data = response.json()

    assert "token" in data
    assert data["token"]

    return data["token"]


def test_register_user(user_data):
    response = requests.post(
        f"{BASE_URL}/register",
        json=user_data,
        timeout=10,
    )

    assert response.status_code == 201, response.text


def test_login_user(registered_user):
    response = requests.post(
        f"{BASE_URL}/login",
        json=registered_user,
        timeout=10,
    )

    assert response.status_code == 200, response.text

    data = response.json()

    assert "token" in data
    assert data["token"]


def get_image_processor_payload():
    image_bytes = IMAGE_PATH.read_bytes()

    image_base64 = base64.b64encode(
        image_bytes
    ).decode("utf-8")

    return {
        "filter": {
            "name": "Negative",
        },
        "image": image_base64,
    }


def create_task(auth_token):
    headers = {
        "Authorization": f"Bearer {auth_token}",
    }

    response = requests.post(
        f"{BASE_URL}/task",
        headers=headers,
        json=get_image_processor_payload(),
        timeout=30,
    )

    assert response.status_code == 201, response.text

    data = response.json()

    assert "task_id" in data
    assert uuid.UUID(data["task_id"])

    return data["task_id"]


def test_create_task(auth_token):
    task_id = create_task(auth_token)

    assert task_id


def test_task_status_and_result(auth_token):
    task_id = create_task(auth_token)

    status_url = f"{BASE_URL}/status/{task_id}"
    result_url = f"{BASE_URL}/result/{task_id}"

    headers = {
        "Authorization": f"Bearer {auth_token}",
    }

    deadline = time.monotonic() + 60

    while time.monotonic() < deadline:
        response = requests.get(
            status_url,
            headers=headers,
            timeout=10,
        )

        assert response.status_code == 200, response.text

        data = response.json()

        assert "status" in data

        if data["status"] == "ready":
            break

        assert data["status"] == "in_progress", (
            f"Undefined status: {data['status']}"
        )

        time.sleep(0.5)

    else:
        pytest.fail("Task is still in progress")

    response = requests.get(
        result_url,
        headers=headers,
        timeout=30,
    )

    assert response.status_code == 200, response.text
    assert response.headers["Content-Type"] == "image/png"
    assert response.content.startswith(b"\x89PNG\r\n\x1a\n")

    with Image.open(io.BytesIO(response.content)) as result:
        result.load()

        assert result.format == "PNG"

        with Image.open(IMAGE_PATH) as original:
            assert result.size == original.size


def test_task_not_found(auth_token):
    invalid_task_id = str(uuid.uuid4())

    status_url = f"{BASE_URL}/status/{invalid_task_id}"
    result_url = f"{BASE_URL}/result/{invalid_task_id}"

    headers = {
        "Authorization": f"Bearer {auth_token}",
    }

    response = requests.get(
        status_url,
        headers=headers,
        timeout=10,
    )

    assert response.status_code == 404, response.text

    response = requests.get(
        result_url,
        headers=headers,
        timeout=10,
    )

    assert response.status_code == 404, response.text


def test_unauthorized_access():
    invalid_task_id = str(uuid.uuid4())

    status_url = f"{BASE_URL}/status/{invalid_task_id}"
    result_url = f"{BASE_URL}/result/{invalid_task_id}"
    task_url = f"{BASE_URL}/task"

    response = requests.post(
        task_url,
        timeout=10,
    )

    assert response.status_code == 401, response.text

    response = requests.get(
        status_url,
        timeout=10,
    )

    assert response.status_code == 401, response.text

    response = requests.get(
        result_url,
        timeout=10,
    )

    assert response.status_code == 401, response.text