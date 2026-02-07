import os
from pathlib import Path

import oss2


def load_env_file(env_path):
    values = {}
    if not env_path.exists():
        return values
    for line in env_path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if "=" not in line:
            continue
        key, value = line.split("=", 1)
        values[key.strip()] = value.strip()
    return values


def main():
    root = Path(__file__).resolve().parent.parent
    env_path = root / "core" / ".env"
    env = load_env_file(env_path)

    access_key = env.get("OSS_ACCESS_KEY_ID", os.getenv("OSS_ACCESS_KEY_ID", ""))
    access_secret = env.get("OSS_ACCESS_KEY_SECRET", os.getenv("OSS_ACCESS_KEY_SECRET", ""))
    bucket_name = env.get("OSS_BUCKET_NAME", os.getenv("OSS_BUCKET_NAME", ""))
    region = env.get("OSS_REGION", os.getenv("OSS_REGION", ""))

    missing = [k for k, v in {
        "OSS_ACCESS_KEY_ID": access_key,
        "OSS_ACCESS_KEY_SECRET": access_secret,
        "OSS_BUCKET_NAME": bucket_name,
        "OSS_REGION": region,
    }.items() if not v]
    if missing:
        raise RuntimeError(f"缺少配置: {', '.join(missing)}")

    endpoint = f"https://{region}.aliyuncs.com"
    auth = oss2.Auth(access_key, access_secret)
    bucket = oss2.Bucket(auth, endpoint, bucket_name)

    print("检查桶信息权限")
    info = bucket.get_bucket_info()
    print("桶信息", info.status)

    key = f"perm-check/{os.getpid()}-{os.urandom(4).hex()}.txt"
    print("检查写入权限")
    put_result = bucket.put_object(key, b"permission-check")
    print("写入结果", put_result.status)

    print("检查读取权限")
    get_result = bucket.get_object(key)
    data = get_result.read()
    print("读取结果", get_result.status, len(data))

    print("检查删除权限")
    del_result = bucket.delete_object(key)
    print("删除结果", del_result.status)


if __name__ == "__main__":
    main()
