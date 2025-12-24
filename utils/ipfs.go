package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// PinataJWT 需要替换为你自己的 Token (生产环境建议放环境变量)
const PinataJWT = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySW5mb3JtYXRpb24iOnsiaWQiOiJlOWM5YWU4YS02OGMwLTQ5MDktOWY5MS01MTg0NWIzNzIyY2IiLCJlbWFpbCI6ImNhcGNoZXIwMTEwQGdtYWlsLmNvbSIsImVtYWlsX3ZlcmlmaWVkIjp0cnVlLCJwaW5fcG9saWN5Ijp7InJlZ2lvbnMiOlt7ImRlc2lyZWRSZXBsaWNhdGlvbkNvdW50IjoxLCJpZCI6IkZSQTEifSx7ImRlc2lyZWRSZXBsaWNhdGlvbkNvdW50IjoxLCJpZCI6Ik5ZQzEifV0sInZlcnNpb24iOjF9LCJtZmFfZW5hYmxlZCI6ZmFsc2UsInN0YXR1cyI6IkFDVElWRSJ9LCJhdXRoZW50aWNhdGlvblR5cGUiOiJzY29wZWRLZXkiLCJzY29wZWRLZXlLZXkiOiJlZGU1ODUwMjNkMGQ5NDU1NzhjNCIsInNjb3BlZEtleVNlY3JldCI6ImIxMTdjZDhmOGFhNWMxNTQ1YTZjMjg0YjBiZGRmNzA2YzNjNzc3ZWJkYjQxNzJkYTczNWJjOGJlODc5MTcwMmMiLCJleHAiOjE3OTgxMDA1Nzh9.Vo-YDMd3VlhFzac2M7XLtT_qx0uobYa0uiC32-SvIm0"

type PinataResponse struct {
	IpfsHash string `json:"IpfsHash"`
}

func UploadToIPFS(filePath string) (string, error) {
	url := "https://api.pinata.cloud/pinning/pinFileToIPFS"

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	io.Copy(part, file)
	writer.Close() // 必须先关闭写入器

	req, _ := http.NewRequest("POST", url, payload)
	req.Header.Add("Authorization", "Bearer "+PinataJWT)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return "", fmt.Errorf("Pinata Error: %d", res.StatusCode)
	}

	var resp PinataResponse
	body, _ := io.ReadAll(res.Body)
	json.Unmarshal(body, &resp)

	return resp.IpfsHash, nil
}
