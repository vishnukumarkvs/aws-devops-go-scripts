# AWS DevOps Scripts

## startstop

- This script contains code that can startstop the AWS resources based on time specified
- Currently supports RDS Instance, ECS Services


## Other Info

Start go app as a systemd service 

```
#!/bin/bash

SERVICE_NAME="myapp"
EXECUTABLE_PATH="/home/ec2-user/myapp"

# Create a systemd service file
echo "[Unit]
Description=MyApp Service
After=network.target

[Service]
Type=simple
User=ec2-user
WorkingDirectory=/home/ec2-user
ExecStart=$EXECUTABLE_PATH
Restart=on-failure

[Install]
WantedBy=multi-user.target" | sudo tee /etc/systemd/system/$SERVICE_NAME.service

# Reload systemd to apply new service
sudo systemctl daemon-reload

# Start the service
sudo systemctl start $SERVICE_NAME

# Enable the service to start on boot
sudo systemctl enable $SERVICE_NAME

echo "$SERVICE_NAME service setup and started."
```

Check logs of systemd service

```
journalctl -u service_name.service
journalctl -fu service_name.service // live
```