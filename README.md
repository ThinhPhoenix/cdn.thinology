  cdn.thinology

cdn.thinology
============

`cdn.thinology` is a lightweight Content Delivery Network (CDN) that leverages the Telegram Bot API for file storage and retrieval. This project provides a cost-effective solution for hosting files and serving them via a CDN-like infrastructure using Telegram's robust backend.

**Author:** Lai Chi Thinh (ThinhPhoenix) - FPT University.

Features
--------

*   **File Upload:** Upload files to Telegram Bot API and receive a unique file URL for accessing the content.
*   **File Metadata Retrieval:** Retrieve file URLs and metadata (such as file size) using dedicated API endpoints.
*   **CORS Support:** Seamless integration with web applications through Cross-Origin Resource Sharing configuration.

Pros
----

*   **Unlimited Storage:** Utilizes Telegram's cloud storage for files, offering virtually unlimited capacity.
*   **Easy to Use:** Integration with Telegram Bot API simplifies file upload and retrieval operations.
*   **Free:** No additional cost for storage or bandwidth usage beyond what Telegram charges for bot API usage.

Cons
----

*   **Automatic Removal:** Files may be removed if Telegram considers them inactive for a prolonged period due to privacy policies and storage management.

Getting Started
---------------

### Prerequisites

*   Go programming language (version 1.16 or later)
*   Telegram Bot API token (obtain one by creating a new bot using BotFather)

### Installation

1.  **Clone the repository:**
```bash
git clone https://github.com/ThinhPhoenix/telikou_cdn.git
cd telikou_cdn
```    

3.  **Install dependencies:**
```golang
go get github.com/gin-contrib/cors
go get github.com/gin-gonic/gin
go get github.com/google/uuid
go get github.com/joho/godotenv
```       
5.  **Run the project:**
```golang
go run main.go
```      

The server will start running on [http://localhost:8080](http://localhost:8080) by default.

Usage
-----

### Upload a File

To upload a file, send a POST request to the `/send` endpoint with the following form data:

*   `bot_token`: Your Telegram Bot API token
*   `chat_id`: The chat ID where you want to upload the file (you can use your own chat ID or a group/channel ID)
*   `document`: The file you want to upload

Example:

    curl -X POST -F "bot_token=<your_bot_token>" -F "chat_id=<your_chat_id>" -F "document=@/path/to/your/file" http://localhost:8080/send
      

### Get File URL

To retrieve the file URL, send a GET request to the `/url` endpoint with the following query parameters:

*   `bot_token`: Your Telegram Bot API token
*   `file_id`: The file ID obtained from the upload response

Example:

    curl -X GET "http://localhost:8080/url?bot_token=<your_bot_token>&file_id=<your_file_id>"
      

### Retrieve File

You can download the file by accessing the secure URL generated after uploading:

*   `/drive/:id`: Endpoint to download the file associated with `:id` (secure ID).

Example:

    curl -OJL http://localhost:8080/drive/<secure_id>
      

### Get File Information

To get information about a file (including its size and URL), send a GET request to the `/info` endpoint with the following query parameters:

*   `bot_token`: Your Telegram Bot API token
*   `file_id`: The file ID obtained from the upload response

Example:

    curl -X GET "http://localhost:8080/info?bot_token=<your_bot_token>&file_id=<your_file_id>"
      

Feel free to adjust the endpoints and placeholders (`<your_bot_token>`, `<your_chat_id>`, `<your_file_id>`, `<secure_id>`) with actual values as per your application's requirements.
