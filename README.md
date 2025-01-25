# OneDrive Upload API

A serverless API service for uploading files to OneDrive with support for multiple remote configurations.

## File Handling & Limits

### Capabilities
- Maximum file size: 5GB
- Binary file upload (supports all file types)
- Chunk sizes: 2MB to 32MB
- Parallel uploads: Up to 4 chunks
- Progress tracking

### Timeouts
- Read timeout: 10 minutes
- Write timeout: 10 minutes
- Idle timeout: 2 minutes

## API Endpoint

### POST /upload

Upload any file to OneDrive as binary data.

**Headers:**
```
Content-Type: application/octet-stream
X-Remote: [remote name] (required) - One of: hakimionedrive, oned, saurajcf
X-Filename: [filename] (required) - Name for the uploaded file
X-Remote-Folder: [folder path] (optional) - Target folder in OneDrive
X-Chunk-Size: [size in MB] (required) - Chunk size (2-32)
```

**Request Body:**
- Raw binary file content

**Example using cURL:**
```bash
curl -X POST \
  -H "Content-Type: application/octet-stream" \
  -H "X-Remote: oned" \
  -H "X-Filename: example.pdf" \
  -H "X-Remote-Folder: documents" \
  -H "X-Chunk-Size: 8" \
  --data-binary "@/path/to/file.pdf" \
  http://localhost:8080/upload
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

**Error Response:**
```json
{
    "error": "Error message here",
    "details": "Detailed error information"
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
- Memory limits: 2GB max, 512MB reserved
- File descriptor limits: 65536
- Temporary file storage mapping
- Configurable timeouts
- Automatic restart on failure
- Timezone configuration

To stop the service:
```bash
docker-compose down
```

### Local Development

1. Clone the repository and add your rclone.conf file

2. Start the development server
```bash
go run main.go
```

## Performance Optimization

### Chunk Size Selection Guidelines
- Small files (< 100MB): 2-4MB chunks
- Medium files (100MB - 1GB): 8-16MB chunks
- Large files (> 1GB): 16-32MB chunks

### Upload Performance
- Parallel processing: 4 chunks simultaneously
- Progress tracking for large files
- Automatic retry on failures
- Memory-efficient binary handling

## Error Handling and Security

### Error Handling
- Detailed error messages
- Upload progress tracking
- Automatic cleanup of temporary files
- Comprehensive request validation

### Security Features
- Binary file handling (no file type restrictions)
- Request size validation
- Memory usage controls
- Temporary file cleanup
- Read-only configuration mounting

## Configuration

The API supports multiple OneDrive remotes:

```go
var rootFolders = map[string]string{
    "hakimionedrive": "Public",
    "oned":           "",
    "saurajcf":       "MY_BOMT_STUFFS",
}

var baseURLs = map[string]string{
    "hakimionedrive": "https://onedrive-vercel-index-kohl-eight-30.vercel.app",
    "oned":           "https://index.sauraj.eu.org",
    "saurajcf":       "https://my-index-azure.vercel.app",
}
```

## Environment Variables

```bash
# Server configuration
SERVER_READ_TIMEOUT=600s    # For handling large file uploads
SERVER_WRITE_TIMEOUT=600s   # For handling large responses
SERVER_IDLE_TIMEOUT=120s    # Connection idle timeout
SERVER_ADDR=0.0.0.0:8080   # Server binding address
```

## Advantages of Binary Upload

1. Efficiency:
   - No multipart form overhead
   - Direct binary data handling
   - Reduced memory usage
   - Better performance for large files

2. Compatibility:
   - Works with any file type
   - No content-type restrictions
   - Consistent handling of all files

3. Simplicity:
   - Straightforward API
   - Simple client implementation
   - Clear error handling

## License

See [LICENSE](LICENSE) file for details.
