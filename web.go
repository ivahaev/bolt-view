package boltview

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/gin-gonic/contrib/gzip"
	"github.com/gin-gonic/gin"
	"net/http"
)

var DB *bolt.DB

var (
	DbNotFound = errors.New("Not found")
)

func Init(db *bolt.DB, _port ...string) {
	DB = db

	router := gin.Default()
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.GET("/", bucketList)
	router.GET("/rest/:bucket/:action", restGetHandler)
	router.POST("/rest", restPostHandler)
	router.GET("/bucket/:name", bucketContent)

	port := "3333"
	if len(_port) > 0 && _port[0] != "" {
		port = _port[0]
	}
	err := http.ListenAndServe(":"+port, router)
	if err != nil {
		panic(err)
	}
}

func restGetHandler(c *gin.Context) {
	bucket := c.Param("bucket")
	if bucket == "" {
		c.JSON(400, "Missing bucket in query")
		return
	}

	action := c.Param("action")
	switch action {
	case "all":
		d, err := getAll(bucket)
		if err != nil {
			c.JSON(400, err)
			return
		}
		var result []interface{}
		for _, _d := range d {
			var _result = map[string]interface{}{}
			err = json.Unmarshal(_d, &_result)
			if err != nil {
				c.JSON(500, err)
				return
			}
			result = append(result, _result)
		}
		c.JSON(200, result)
		return
	case "one":
		id := c.Query("id")
		if id != "" {
			d, err := get(bucket, id)
			if err != nil {
				c.JSON(400, err)
				return
			}
			var result interface{}
			err = json.Unmarshal(d, &result)
			if err != nil {
				c.JSON(500, err)
				return
			}
			c.JSON(200, result)
		}
	}
}

func restPostHandler(c *gin.Context) {

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

func delete(bucket, key string) (err error) {
	DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			err = DbNotFound
			return err
		}
		err = b.Delete([]byte(key))
		return err
	})
	return
}

func get(bucket, key string) (result []byte, err error) {
	DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			err = DbNotFound
			return err
		}
		v := b.Get([]byte(key))

		if v == nil {
			err = DbNotFound
			return err
		}
		result = append(result, v...)

		return nil
	})
	return
}

func getAll(bucket string) (result [][]byte, err error) {
	DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			err = DbNotFound
			return err
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			result = append(result, v)
		}
		return nil
	})
	return
}

func newUUIDv4() string {
	u := [16]byte{}
	_, err := rand.Read(u[:16])
	if err != nil {
		panic(err)
	}

	u[8] = (u[8] | 0x80) & 0xBf
	u[6] = (u[6] | 0x40) & 0x4f

	return fmt.Sprintf("%x-%x-%x-%x-%x", u[:4], u[4:6], u[6:8], u[8:10], u[10:])
}

func set(bucket, key string, value []byte) (err error) {
	DB.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		err = b.Put([]byte(key), value)
		return err
	})
	return
}
