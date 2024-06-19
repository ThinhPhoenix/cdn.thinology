# telikou_cdn

telikou_cdn is a Content Delivery Network (CDN) system that utilizes the Telegram Bot API as a storage solution for files. This project provides a lightweight and cost-effective way to host and serve files by leveraging the Telegram infrastructure.

## Features

- Upload files to the Telegram Bot API and receive a unique file URL for accessing the content
- Retrieve file URLs and metadata (file size, etc.) using the provided API endpoints
- Cross-Origin Resource Sharing (CORS) support for seamless integration with web applications

## Getting Started

### Prerequisites

- Go programming language (version 1.16 or later)
- Telegram Bot API token (obtain one by creating a new bot using the BotFather)

### Installation

1. Clone the repository:

    ```bash
    git clone https://github.com/ThinhPhoenix/telikou_cdn.git
    ```

2. Navigate to the project directory:

    ```bash
    cd telikou_cdn
    ```

3. Install the required dependencies:

    ```go
    go get github.com/gin-contrib/cors
    go get github.com/gin-gonic/gin
    go get github.com/joho/godotenv
    ```

4. Run the project:

    ```go
    go run ./telikou_cdn
    ```

    The server will start running on [http://localhost:8080](http://localhost:8080) by default.

### Usage

#### Upload a File

To upload a file, send a POST request to the `/upload` endpoint with the following form data:

- `bot_token`: Your Telegram Bot API token
- `chat_id`: The chat ID where you want to upload the file (you can use your own chat ID or a group/channel ID)
- `document`: The file you want to upload

#### Get File URL

To retrieve the file URL, send a GET request to the `/url` endpoint with the following query parameters:

- `bot_token`: Your Telegram Bot API token
- `file_id`: The file ID obtained from the upload response
