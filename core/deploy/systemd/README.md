`cloud-disk-backend.service` is a `systemd` unit for the backend process on the remote host.

Install on the server:

```bash
cp deploy/systemd/cloud-disk-backend.service /etc/systemd/system/cloud-disk-backend.service
cp deploy/systemd/cloud-disk-backend.env.example /home/cloud_disk/backend/cloud-disk-backend.env
chmod 600 /home/cloud_disk/backend/cloud-disk-backend.env
systemctl daemon-reload
systemctl enable --now cloud-disk-backend
```

Useful commands:

```bash
systemctl status cloud-disk-backend
systemctl restart cloud-disk-backend
journalctl -u cloud-disk-backend -f
tail -f /home/cloud_disk/backend/go-backend.log
```
