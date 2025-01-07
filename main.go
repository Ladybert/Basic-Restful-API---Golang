package main

import (
	"net/http"
    "github.com/gin-gonic/gin"
	 "database/sql"
    _ "github.com/go-sql-driver/mysql"
	"fmt"
	"log"
	"encoding/hex"
	"math/rand"
)

type album struct {
    ID     string  `json:"id"`
    Title  string  `json:"title"`
    Artist string  `json:"artist"`
    Price  float64 `json:"price"`
}

func main() {
	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3360)/golangpractice")
	if err != nil {
		log.Fatalf("Error saat membuka koneksi database: %v", err)
	}
	defer db.Close()

	// Ping untuk memastikan koneksi berhasil
	if err := db.Ping(); err != nil {
		log.Fatalf("Koneksi database gagal: %v", err)
	}

	fmt.Println("Koneksi ke database berhasil!")

    router := gin.Default()
    router.GET("/albums", withDB(db, getAlbums))
	router.GET("/albums/:id", withDB(db, getAlbumByID))
	router.POST("/albums", withDB(db, postAlbum))
	router.PUT("/albums/:id", withDB(db, updateAlbum))
	router.DELETE("/albums/:id", withDB(db, deleteAlbum))

    router.Run("localhost:3000")
}

func withDB(db *sql.DB, handler func(*gin.Context, *sql.DB)) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(c, db)
	}
}

func getAlbums(c *gin.Context, db *sql.DB) {
	rows, err := db.Query("SELECT * FROM albums")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error querying database"})
		return
	}
	defer rows.Close()

	var albums []album

	for rows.Next() {
		var a album
		if err := rows.Scan(&a.ID, &a.Title, &a.Artist, &a.Price); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning row"})
			return
		}
		albums = append(albums, a)
	}

	// Pastikan tidak ada error pada rows
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating rows"})
		return
	}

	// Return data sebagai JSON
	c.IndentedJSON(http.StatusOK, albums)
}

func getAlbumByID(c *gin.Context, db *sql.DB) {
	id := c.Param("id") // Ambil parameter ID dari URL

	// Query database untuk mencari album berdasarkan ID
	query := "SELECT * FROM albums WHERE id = ?"
	var album album
	err := db.QueryRow(query, id).Scan(&album.ID, &album.Title, &album.Artist, &album.Price)

	if err != nil {
		if err == sql.ErrNoRows {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
		} else {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve album"})
		}
		return
	}

	// Jika ditemukan, kirimkan album ke response
	c.IndentedJSON(http.StatusOK, album)
}

func postAlbum(c *gin.Context, db *sql.DB) {
	var newAlbum album

	// Bind JSON dari request body ke struct, tanpa ID
	if err := c.BindJSON(&newAlbum); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Generate ID unik secara otomatis
	id, err := generateUniqueID(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate unique ID"})
		return
	}
	newAlbum.ID = id // Set ID yang digenerate ke struct

	// Simpan data ke database
	query := "INSERT INTO albums (id, title, artist, price) VALUES (?, ?, ?, ?)"
	_, err = db.Exec(query, newAlbum.ID, newAlbum.Title, newAlbum.Artist, newAlbum.Price)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert album into database"})
		return
	}

	// Return album yang berhasil disimpan
	c.IndentedJSON(http.StatusCreated, newAlbum)
}

func updateAlbum(c *gin.Context, db *sql.DB) {
	id := c.Param("id")
	var updatedAlbum album

	// Bind JSON dari request body ke struct album
	if err := c.BindJSON(&updatedAlbum); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Query untuk memperbarui album berdasarkan ID
	query := "UPDATE albums SET title = ?, artist = ?, price = ? WHERE id = ?"
	result, err := db.Exec(query, updatedAlbum.Title, updatedAlbum.Artist, updatedAlbum.Price, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update album"})
		return
	}

	// Periksa apakah ada baris yang terpengaruh
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch affected rows"})
		return
	}
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "album not found"})
		return
	}

	// Tampilkan data yang telah diperbarui sebagai respons
	updatedAlbum.ID = id
	c.IndentedJSON(http.StatusOK, updatedAlbum)
}


func deleteAlbum(c *gin.Context, db *sql.DB) {
	id := c.Param("id")

	query := "DELETE FROM albums WHERE id = ?"
	result, err := db.Exec(query, id)

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "failed to delete album"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "failed to check delete operation"})
		return
	}

	if rowsAffected == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"message": "album deleted"})
}


func generateUniqueID(db *sql.DB) (string, error) {
	for {
		// Buat ID acak dengan prefix "album-"
		randomBytes := make([]byte, 8) // 8 byte -> 16 karakter hex
		rand.Read(randomBytes)
		newID := "album-" + hex.EncodeToString(randomBytes)

		// Cek apakah ID sudah ada di database
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM albums WHERE id = ?)", newID).Scan(&exists)
		if err != nil {
			return "", err
		}

		// Jika ID belum ada, return ID
		if !exists {
			return newID, nil
		}
	}
}