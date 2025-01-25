# OneDrive Upload API

A serverless API service for uploading files to OneDrive with support for multiple remote configurations.

## API Endpoint

### POST /upload

Upload a file to OneDrive.

**Request Format:**
- Content-Type: `multipart/form-data`

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

### Prerequisites
1. OneDrive API credentials in rclone.conf
2. Vercel account and Vercel CLI installed

### Deploy to Vercel

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

### Local Development

1. Clone the repository and add your rclone.conf file

2. Start the development server
```bash
go run main.go
```

The server will start on http://localhost:8080 with the following endpoint:
- Upload endpoint: http://localhost:8080/upload

## Configuration

The API supports multiple OneDrive remotes which are configured in the code:

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

## Error Handling and Logging

The API includes comprehensive error handling and logging:
- Request validation with detailed error messages
- File size and type validation
- Detailed logging of upload process
- Proper cleanup of temporary files
- Memory limits to prevent server overload

## Security

- The API uses CORS headers to allow requests from any origin
- Request size limits and validations are in place
- Proper cleanup of temporary files
- Keep your rclone.conf secure and never commit it to version control

## License

See [LICENSE](LICENSE) file for details.
