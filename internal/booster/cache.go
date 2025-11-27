package booster

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imroc/req/v3"
	"github.com/ssdomei232/faster/handler/db"
)

// TODO: 特别加速高热资源(S3+302)
func HandleHttpRequest(c *gin.Context) {
	var err error

	url := strings.TrimPrefix(c.Param("url"), "/")
	urlHash := getUrlHash(url)

	isCached, err := checkCache(urlHash)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"code": 500,
			"msg":  "check cache failed",
		})
		return
	}

	// cache file path rule is "data/<url_hash>.<file extension>"
	// so that we can use url_hash to find cache file
	cacheFilepath := "data/" + urlHash + getFileExtensionFromURL(url)

	// Do response
	if isCached {
		isCacheExpired, err := checkCacheExpired(url)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{
				"code": 500,
				"msg":  "check cache expire failed",
			})
			return
		}

		if isCacheExpired {
			err = refreshCache(url)
			if err != nil {
				c.AbortWithStatusJSON(500, gin.H{
					"code": 500,
					"msg":  "refresh cache failed",
				})
				return
			}
		}

		c.File(cacheFilepath)
	} else {
		if err = cacheFile(url); err != nil {
			c.AbortWithStatusJSON(500, gin.H{
				"code": 500,
				"msg":  "cache file failed",
			})
			return
		} else {
			c.File(cacheFilepath)
		}
	}
}

// cache File and insert into db
func cacheFile(url string) error {
	db, err := db.GetDB()
	if err != nil {
		return err
	}
	defer db.Close()

	// download file
	urlHash := getUrlHash(url)
	req.R().SetOutputFile("data/" + urlHash + getFileExtensionFromURL(url)).Get(url)

	// insert into db
	_, err = db.Exec("INSERT INTO file (url_raw, url_hash, exp_at) VALUES (?, ?, ?)", url, urlHash, time.Now().Add(time.Hour*24*7).Unix())

	return err
}

// refresh cache and update db
func refreshCache(url string) error {
	db, err := db.GetDB()
	if err != nil {
		return err
	}
	defer db.Close()

	urlHash := getUrlHash(url)
	cacheFilepath := "data/" + urlHash + getFileExtensionFromURL(url)

	// delete old file from data dir
	err = os.Remove(cacheFilepath)
	if err != nil {
		return err
	}

	// download file
	req.R().SetOutputFile("data/" + urlHash + getFileExtensionFromURL(url)).Get(url)

	// update db
	_, err = db.Exec("UPDATE urls SET exp_at = ? WHERE url_hash = ?", time.Now().Add(time.Hour*24*7).Unix(), urlHash)

	return err
}

// check whwather cache file expired
func checkCacheExpired(url string) (bool, error) {
	db, err := db.GetDB()
	if err != nil {
		return false, err
	}
	defer db.Close()

	var expAt int64
	urlHash := getUrlHash(url)
	err = db.QueryRow("SELECT exp_at FROM file WHERE url_hash = ?", urlHash).Scan(&expAt)
	if err != nil {
		return false, err
	}

	return time.Now().Unix() > expAt, nil
}

// check wheather url is cached, if yes, return true, else return false
func checkCache(urlHash string) (isExist bool, err error) {
	db, err := db.GetDB()
	if err != nil {
		return false, err
	}
	defer db.Close()

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM file WHERE url_hash = ? LIMIT 1)", urlHash).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// get url sha256
func getUrlHash(url string) string {
	h := sha256.New()
	h.Write([]byte(url))
	bs := h.Sum(nil)
	return hex.EncodeToString(bs)
}

// get file extension from url
func getFileExtensionFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return path.Ext(u.Path)
}
