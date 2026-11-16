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


@pytest.fixture(scope="module")
def auth_headers():
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


def wait_for_task(task_id, headers, timeout=60):
    deadline = time.monotonic() + timeout

    while time.monotonic() < deadline:
        response = requests.get(
            f"{BASE_URL}/status/{task_id}",
            headers=headers,
            timeout=5,
        )

        assert response.status_code == 200, response.text

        status = response.json()["status"]

        if status == "ready":
            return status

        if status == "failed":
            pytest.fail(f"Task {task_id} failed")

        assert status == "in_progress", f"Unexpected status: {status}"

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
    image_base64 = create_test_image()

    filter_data = {
        "name": filter_name,
    }

    if parameters is not None:
        filter_data["parameters"] = parameters

    payload = {
        "image": image_base64,
        "filter": filter_data,
    }

    response = requests.post(
        f"{BASE_URL}/task",
        headers=auth_headers,
        json=payload,
        timeout=20,
    )

    assert response.status_code == 201, response.text

    task_id = response.json()["task_id"]

    assert task_id

    status = wait_for_task(task_id, auth_headers)

    assert status == "ready"

    result_response = requests.get(
        f"{BASE_URL}/result/{task_id}",
        headers=auth_headers,
        timeout=30,
    )

    assert result_response.status_code == 200, result_response.text

    assert result_response.headers["Content-Type"] == "image/png"

    assert result_response.content.startswith(b"\x89PNG\r\n\x1a\n")

    image = Image.open(io.BytesIO(result_response.content))
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