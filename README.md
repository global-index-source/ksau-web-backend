# OneDrive Upload API

A serverless API service for uploading files to OneDrive with support for multiple remote configurations.

## File Size Limits & Performance

### Default Limits
- Maximum file size: 1GB
- In-memory parsing limit: 100MB
- Chunk size options: 2MB to 16MB
- Parallel uploads: Up to 4 chunks simultaneously

### Configurable Timeouts
- Read timeout: 10 minutes
- Write timeout: 10 minutes
- Idle timeout: 2 minutes

### Environment Variables
```bash
# Server configuration
SERVER_READ_TIMEOUT=600s    # For handling large file uploads
SERVER_WRITE_TIMEOUT=600s   # For handling large file responses
SERVER_IDLE_TIMEOUT=120s    # Connection idle timeout
SERVER_ADDR=0.0.0.0:8080   # Server binding address
```

## API Endpoint

### POST /upload

Upload a file to OneDrive.

**Request Format:**
- Content-Type: `multipart/form-data`
- Maximum file size: 1GB

**Form Fields:**
- `file`: (required) The file to upload
- `remote`: (required) Remote configuration to use (`hakimionedrive`, `oned`, or `saurajcf`)
- `remoteFolder`: (optional) Folder path in OneDrive where the file should be uploaded
- `remoteFileName`: (optional) Custom name for the uploaded file
- `chunkSize`: (required) Upload chunk size in MB (2, 4, 8, or 16)

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

### Option 1: Docker Deployment

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

### Option 2: Vercel Deployment

Note: Vercel has its own limitations:
- Maximum file size: 50MB
- Execution timeout: 10 seconds
- Memory: 1024MB

1. Clone this repository
```bash
git clone https://github.com/ksauraj/ksau-oned-api.git
cd ksau-oned-api
```

2. Add your rclone.conf file to the project root

3. Deploy to Vercel
```bash
vercel
```

### Option 3: Local Development

1. Clone the repository and add your rclone.conf file

2. Start the development server
```bash
go run main.go
```

## Performance Optimization

### Chunk Size Selection
- Small files (< 100MB): 2MB chunks
- Medium files (100MB - 500MB): 4MB or 8MB chunks
- Large files (> 500MB): 16MB chunks

### Parallel Upload
- The API uses up to 4 parallel chunks for faster uploads
- Automatically manages memory usage
- Includes progress tracking

### Memory Management
- Efficient temp file handling
- Automatic cleanup
- Progress monitoring
- Memory limit enforcement

## Error Handling and Logging

The API includes comprehensive error handling and logging:
- Request validation with detailed error messages
- File size and type validation
- Upload progress tracking
- Detailed logging of upload process
- Proper cleanup of temporary files
- Memory limits to prevent server overload

## Security

- The API uses CORS headers to allow requests from any origin
- Request size limits and validations
- Proper cleanup of temporary files
- Memory usage limits
- Keep your rclone.conf secure and never commit it to version control
- Docker deployment uses read-only mount for rclone.conf

## Limitations

1. File Size:
   - Maximum file size: 1GB (configurable)
   - Memory parsing limit: 100MB
   - Vercel deployment limited to 50MB

2. Timeouts:
   - Default read/write timeout: 10 minutes
   - Can be extended via environment variables

3. Resources:
   - Memory usage depends on chunk size and parallel uploads
   - Temporary storage needed for file processing
   - Network bandwidth affects upload speed

4. Rate Limiting:
   - OneDrive API has its own rate limits
   - Consider implementing rate limiting for production use

## License

See [LICENSE](LICENSE) file for details.
