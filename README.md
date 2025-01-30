# OneDrive Upload API

A serverless API service for uploading files to OneDrive with support for multiple remote configurations.

## API Endpoints

### 1. POST /upload

Upload any file to OneDrive as binary data.

**Headers:**
```
Content-Type: application/octet-stream
X-Remote: [remote name] (required) - One of: hakimionedrive, oned, saurajcf
X-Filename: [filename] (required) - Name for the uploaded file
X-Remote-Folder: [folder path] (optional) - Target folder in OneDrive
X-Chunk-Size: [size in MB] (required) - Chunk size (2-32)
```

**Success Response:**
```json
{
    "status": "success",
    "message": "File uploaded successfully",
    "downloadURL": "https://your-index.vercel.app/path/to/file",
    "fileSize": 1234567,
    "fileName": "example.pdf"
}
```

### 2. GET /system

Get basic system information.

**Success Response:**
```json
{
    "status": "success",
    "data": {
        "cpu": {
            "model": "AMD Ryzen 9 5900X",
            "cores": 12,
            "usage": 25.5,
            "load_percentage": ["10%", "15%", "25%"]
        },
        "memory": {
            "total": 34359738368,
            "used": 8589934592,
            "free": 25769803776,
            "used_percent": 25.0
        },
        "system": {
            "hostname": "server-name",
            "os": "linux",
            "platform": "ubuntu",
            "kernel": "5.15.0-1059",
            "architecture": "amd64",
            "server_time": "2025-01-26T02:35:00+05:30",
            "uptime": 1234567
        }
    }
}
```

### 3. GET /neofetch

Get detailed system information in a neofetch-like format with ASCII art and styling.

**Success Response:**
```json
{
    "status": "success",
    "data": {
        "ascii_art": "...[ASCII art here]...",
        "colors": {
            "primary": "[38;2;0;255;0m",
            "secondary": "[38;2;0;200;0m",
            "accent": "[38;2;50;255;50m"
        },
        "system": {
            "user": "username@hostname",
            "hostname": "server-name",
            "distro": "Ubuntu 22.04 LTS",
            "kernel": "5.15.0-1059",
            "uptime": "10 days, 5 hours, 30 minutes",
            "shell": "/bin/bash",
            "cpu": "AMD Ryzen 9 5900X (12) @ 3.70GHz",
            "memory": "8.0 GiB / 32.0 GiB (25.0%)",
            "disk_usage": "100.0 GiB / 500.0 GiB (20.0%)",
            "local_ip": "192.168.1.100",
            "server_time": "2025-01-26T02:35:00+05:30",
            "load_average": [1.5, 1.2, 1.0]
        },
        "performance": {
            "cpu_usage": 25.5,
            "memory_usage": 25.0,
            "cpu_frequency": 3700,
            "core_loads": [20.5, 15.2, 30.1]
        }
    }
}
```

## Deployment

### System Requirements

#### Minimum Requirements
- RAM: 512MB
- Storage: Depends on upload sizes (temporary storage)
- File descriptors: 65536
- Network: Stable internet connection

#### Recommended
- RAM: 2GB or more
- Storage: 10GB+ for temporary files
- CPU: 2 cores or more
- Network: High-speed internet connection

### Docker Deployment

1. Clone the repository
```bash
git clone https://github.com/ksauraj/ksau-oned-api.git
cd ksau-oned-api
```

2. Add your rclone.conf to the project root

3. Deploy using Docker Compose:
```bash
docker-compose up -d
```

Docker configuration provides:
- Memory limits: 4GB max, 512MB reserved
- File descriptor limits: 65536
- Temporary file storage mapping
- Configurable timeouts
- Automatic restart on failure
- Timezone configuration

### Environment Variables

```bash
# Server configuration
SERVER_READ_TIMEOUT=1800s    # For handling large files
SERVER_WRITE_TIMEOUT=1800s   # For handling large responses
SERVER_IDLE_TIMEOUT=120s     # Connection idle timeout
SERVER_ADDR=0.0.0.0:8080    # Server binding address
```

## Performance Optimization

### Upload Optimization
- Configurable chunk sizes (2-32MB)
- Sequential chunk processing for reliability
- Progress tracking
- Automatic retry on failures

### System Information
- Cached responses for system info
- Real-time CPU and memory monitoring
- Detailed performance metrics
- ASCII art generation for frontend display

## Security Features
- Binary file handling
- Request size validation
- Memory usage controls
- Temporary file cleanup
- Read-only configuration mounting

## License

See [LICENSE](LICENSE) file for details.
