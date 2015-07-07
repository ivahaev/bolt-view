package boltview

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/boltdb/bolt"
	"github.com/gin-gonic/contrib/gzip"
	"github.com/gin-gonic/gin"
	"net/http"
)

var DB *bolt.DB

func Init(db *bolt.DB) {
	DB = db

	router := gin.Default()
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.GET("/", bucketList)
	router.GET("/bucket/:name", bucketContent)
	err := http.ListenAndServe(":3333", router)
	if err != nil {
		panic(err)
	}
}

func bucketList(c *gin.Context) {
	buckets := []string{}
	DB.View(func(tx *bolt.Tx) error {
		tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			buckets = append(buckets, string(name))
			return nil
		})
		return nil
	})
	var html string
	html += `<html><head><meta charset="UTF-8"></head><body>`
	html += `<h1>Bucket list:</h1>`
	for _, b := range buckets {
		html += `<div><a href="/bucket/` + b + `">` + b + `</a></div>`
	}
	html += `</body></html>`
	c.Data(200, "text/html", []byte(html))
}

func bucketContent(c *gin.Context) {
	name := c.Param("name")
	content := map[string]string{}
	DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(name))
		if b == nil {
			return errors.New("Wrong bucket")
		}
		b.ForEach(func(k, v []byte) error {
			var out bytes.Buffer
			json.Indent(&out, v, "", "\t")
			content[string(k)] = out.String()
			return nil
		})

		return nil
	})
	var html string
	html += `<html><head><meta charset="UTF-8"></head><body>`
	html += `<h1>Bucket's "` + name + `" content:</h1>`
	for k, v := range content {
		html += `<div><h2>` + k + `:</h2><pre>` + v + `</pre></div>`
	}
	html += `</body></html>`
	c.Data(200, "text/html", []byte(html))
}
