import hashlib
import os
import tempfile
import time

import requests


def request_json(method, url, token=None, **kwargs):
    headers = kwargs.pop("headers", {})
    if token:
        headers["Authorization"] = f"Bearer {token}"
    resp = requests.request(method, url, headers=headers, timeout=60, **kwargs)
    try:
        payload = resp.json()
    except Exception:
        print("响应状态码", resp.status_code)
        print("响应头", dict(resp.headers))
        print("响应内容", resp.text)
        raise RuntimeError(f"响应非JSON: {resp.status_code} {resp.text}")
    if resp.status_code != 200 or payload.get("code") != 0:
        print("响应状态码", resp.status_code)
        print("响应头", dict(resp.headers))
        print("响应内容", payload)
        raise RuntimeError(f"请求失败: {resp.status_code} {payload}")
    return payload.get("data")


def request_should_fail(method, url, **kwargs):
    resp = requests.request(method, url, timeout=30, **kwargs)
    try:
        payload = resp.json()
    except Exception:
        return
    if resp.status_code == 200 and payload.get("code") == 0:
        raise RuntimeError("鉴权测试失败: 未携带token却成功")


def main():
    base = os.environ.get("BASE_URL", "http://127.0.0.1:8888").rstrip("/")
    username = os.environ.get("TEST_USER", "admin")
    password = os.environ.get("TEST_PASSWORD", "xn5rfaBICYqf")

    print("登录获取token")
    login = request_json(
        "POST",
        f"{base}/api/users/login",
        json={"name": username, "password": password},
    )
    token = login.get("token")
    if not token:
        raise RuntimeError("登录未返回token")

    print("鉴权拦截校验")
    request_should_fail(
        "POST",
        f"{base}/api/file/user/list",
        json={"id": 0, "page": 1, "size": 10},
    )

    suffix = str(int(time.time()))
    filename = f"e2e-{suffix}.txt"
    content = f"e2e-{suffix}".encode("utf-8")
    digest = hashlib.md5(content).hexdigest()

    with tempfile.NamedTemporaryFile(delete=False) as tmp:
        tmp.write(content)
        tmp_path = tmp.name

    print("上传文件")
    upload_failed = False
    with open(tmp_path, "rb") as fh:
        files = {"file": (filename, fh, "text/plain")}
        data = [
            ("parent_id", "0"),
            ("hash", digest),
            ("name", filename),
            ("ext", ".txt"),
            ("size", str(len(content))),
            ("object_key", ""),
            ("ParentId", "0"),
            ("Hash", digest),
            ("Name", filename),
            ("Ext", ".txt"),
            ("Size", str(len(content))),
            ("ObjectKey", ""),
        ]
        try:
            request_json("POST", f"{base}/api/file/upload", token=token, files=files, data=data)
        except RuntimeError as err:
            if "AccessDenied" in str(err) or "bucket acl" in str(err):
                print("上传失败，OSS权限不足，改用已有文件继续测试")
                upload_failed = True
            else:
                raise

    print("获取文件列表")
    list_data = request_json(
        "POST",
        f"{base}/api/file/user/list",
        token=token,
        json={"id": 0, "page": 1, "size": 50},
    )
    items = list_data.get("list") or []
    target = None
    if not upload_failed:
        for item in items:
            if item.get("name") == filename:
                target = item
                break
        if not target:
            raise RuntimeError("未在列表中找到上传文件")
    else:
        if items:
            target = items[0]
        else:
            raise RuntimeError("OSS权限不足且无可用文件，无法继续测试")

    file_identity = target.get("identity")
    repository_identity = target.get("repository_identity")
    if not file_identity or not repository_identity:
        raise RuntimeError("文件标识信息缺失")

    print("下载URL获取")
    request_json(
        "POST",
        f"{base}/api/file/url",
        token=token,
        json={"repository_identity": repository_identity, "expires": 600},
    )

    new_name = f"renamed-{suffix}.txt"
    print("文件重命名")
    request_json(
        "POST",
        f"{base}/api/file/user/file/name/update",
        token=token,
        json={"identity": file_identity, "name": new_name},
    )

    folder_name = f"folder-{suffix}"
    print("创建文件夹")
    folder_data = request_json(
        "POST",
        f"{base}/api/file/user/folder/create",
        token=token,
        json={"parent_id": 0, "name": folder_name},
    )
    print("创建文件夹响应", folder_data)
    folder_identity = folder_data.get("identity")
    folder_id = folder_data.get("id")
    if not folder_identity:
        raise RuntimeError("创建文件夹未返回identity")

    print("解析文件夹ID")
    if not folder_id:
        page = 1
        size = 100
        while True:
            list_data = request_json(
                "POST",
                f"{base}/api/file/user/list",
                token=token,
                json={"id": 0, "page": page, "size": size},
            )
            print("文件列表响应", list_data)
            items = list_data.get("list") or []
            for item in items:
                if item.get("identity") == folder_identity or item.get("name") == folder_name:
                    folder_id = item.get("id")
                    break
            if folder_id or len(items) < size:
                break
            page += 1
    if not folder_id:
        raise RuntimeError("未找到文件夹ID")

    print("移动文件")
    request_json(
        "PUT",
        f"{base}/api/file/user/file/move",
        token=token,
        json={"identity": file_identity, "parent_id": int(folder_id), "name": new_name},
    )

    print("创建分享")
    share_data = request_json(
        "POST",
        f"{base}/api/share/create",
        token=token,
        json={"identity": repository_identity, "expired_time": 600},
    )
    share_identity = share_data.get("identity")
    if not share_identity:
        raise RuntimeError("创建分享未返回identity")

    print("获取分享详情")
    request_json(
        "GET",
        f"{base}/api/share/get",
        token=token,
        params={"identity": share_identity},
    )

    print("分享下载URL获取")
    request_json(
        "POST",
        f"{base}/api/share/url",
        token=token,
        json={"share_identity": share_identity, "expires": 600},
    )

    print("保存分享资源")
    request_json(
        "POST",
        f"{base}/api/share/save",
        token=token,
        json={"repository_identity": repository_identity, "parent_id": 0, "name": f"saved-{new_name}"},
    )

    print("删除文件夹")
    request_json(
        "DELETE",
        f"{base}/api/file/user/folder/delete",
        token=token,
        json={"identity": folder_identity},
    )

    os.remove(tmp_path)
    print(f"完成: {filename} md5={digest}")


if __name__ == "__main__":
    main()
