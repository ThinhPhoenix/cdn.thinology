package models

type FileData struct {
    ID        string `json:"id,omitempty"`
    URL       string `json:"url,omitempty"`
    SecureURL string `json:"secure_url,omitempty"`
    Bytes     int    `json:"bytes,omitempty"`
    Format    string `json:"format,omitempty"`
}