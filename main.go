package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type FileInfo struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Mode      string    `json:"mode"`
	ModTime   time.Time `json:"modTime"`
	IsDir     bool      `json:"isDir"`
	Extension string    `json:"extension"`
}

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		output, err := runCommand(string(p))
		if err != nil {
			output = []byte("Error: " + err.Error())
		}

		if err := conn.WriteMessage(messageType, output); err != nil {
			log.Println(err)
			return
		}
	}
}

func main() {
	r := gin.Default()

	r.GET("/ws", handleWebSocket)

	r.GET("/", func(c *gin.Context) {
		c.File("index.html")
	})

	r.GET("/ws/getdir", func(c *gin.Context) {
		path := c.DefaultQuery("path", ".")
		absolutePath, err := filepath.Abs(path)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		files, err := listDir(absolutePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, files)
	})

	server := &http.Server{
		Addr:    ":8443",
		Handler: r,
	}

	// Run the server with TLS
	log.Println("Server is running on https://localhost:8443")
	if err := server.ListenAndServeTLS("certificates/localhost.pem", "certificates/localhost-key.pem"); err != nil && err != http.ErrServerClosed {
		log.Fatal("Failed to run server: ", err)
	}
}

func listDir(path string) ([]FileInfo, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var fileList []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			log.Printf("Error getting info for %s: %v", entry.Name(), err)
			continue
		}

		fileInfo := FileInfo{
			Name:      info.Name(),
			Size:      info.Size(),
			Mode:      info.Mode().String(),
			ModTime:   info.ModTime(),
			IsDir:     info.IsDir(),
			Extension: filepath.Ext(info.Name()),
		}

		fileList = append(fileList, fileInfo)
	}

	return fileList, nil
}

func runCommand(command string) ([]byte, error) {
	cmd := exec.Command("sh", "-c", command)
	return cmd.CombinedOutput()
}
