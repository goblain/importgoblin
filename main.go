package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rwcarlsen/goexif/exif"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type importConfig struct {
	from   string
	to     string
	db     string
	delete bool
	force  bool
}

var ic importConfig

var imageExtensions = map[string]string{
	".jpg":  "jpg",
	".jpeg": "jpg",
	".png":  "png",
}

var db *sql.DB

func getFilesToProcess() []string {
	files := []string{}
	filepath.Walk(ic.from, func(filepath string, f os.FileInfo, err error) error {
		ext := path.Ext(strings.ToLower(filepath))
		if _, ok := imageExtensions[ext]; ok {
			files = append(files, filepath)
		}
		return nil
	})
	return files
}

func copy(srcFile, dstFile string) error {
	srcStat, err := os.Stat(srcFile)
	if err != nil {
		return err
	}
	if !srcStat.Mode().IsRegular() {
		return fmt.Errorf("can not copy non-regular source file %s (%q)", srcStat.Name(), srcStat.Mode().String())
	}
	dstStat, err := os.Stat(dstFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		return fmt.Errorf("destination already exists %s", dstStat.Name())
	}
	src, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer src.Close()
	dstDir := path.Dir(dstFile)
	err = os.MkdirAll(dstDir, 0700)
	if err != nil {
		return err
	}
	log.Error(dstDir)
	dst, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}
	dst.Sync()
	err = os.Chtimes(dstFile, srcStat.ModTime(), srcStat.ModTime())
	if err != nil {
		return err
	}
	return nil
}

func processFile(file string) error {
	log.WithField("file", file).Debug("Processing")
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return err
	}

	f.Seek(0, 0)

	var date *time.Time
	ext := path.Ext(strings.ToLower(file))
	if imageExtensions[ext] == "jpg" {
		meta, err := exif.Decode(f)
		if err != nil {
			return err
		}
		taken, err := meta.DateTime()
		if err != nil {
			return err
		}
		date = &taken
		log.WithField("file", file).Debugf("EXIV Taken: %v", taken)
	}

	if date == nil {
		fileStat, _ := os.Stat(file)
		taken := fileStat.ModTime()
		date = &taken
	}
	datetime := fmt.Sprintf("%d%02d%02d%02d%02d%02d", date.Year(), date.Month(), date.Day(), date.Hour(), date.Minute(), date.Second())
	md5sum := hex.EncodeToString(hasher.Sum(nil))
	if ic.force || !wasProcessed(datetime, md5sum) {
		dir := fmt.Sprintf("%04d/%02d/%02d", date.Year(), date.Month(), date.Day())
		dest := fmt.Sprintf("%s/%s/%s_%s%s", ic.to, dir, datetime, md5sum, ext)
		log.WithField("file", file).Infof("copy to %s", dest)
		err = copy(file, dest)
		if err != nil {
			log.WithField("file", file).Errorf("Copy failed with %s", err.Error())
		}
		markProcessed(datetime, md5sum)
	} else {
		log.WithField("file", file).Info("Already processed before. SKIP")
	}
	return nil
}

func wasProcessed(datetime, hash string) bool {
	var count int
	rows, err := db.Query("SELECT count(*) FROM processed WHERE time = ? AND hash = ?", datetime, hash)
	if err != nil {
		log.Error(err.Error())
	}
	defer rows.Close()
	rows.Next()
	rows.Scan(&count)
	if count > 0 {
		return true
	}
	return false
}

func markProcessed(datetime, hash string) error {
	_, err := db.Exec("INSERT INTO processed VALUES (?, ?)", datetime, hash)
	if err != nil {
		log.Error(err.Error())
	}
	return nil
}

func dbInit() {
	var err error
	os.MkdirAll(path.Base(ic.db), 0700)
	db, err = sql.Open("sqlite3", ic.db)
	if err != nil {
		log.Error(err.Error())
		panic(err.Error())
	}
	stmt, err := db.Prepare("CREATE TABLE IF NOT EXISTS processed (time text, hash text)")
	if err != nil {
		log.Error(err.Error())
		panic(err.Error())
	}
	stmt.Exec()
	stmt, err = db.Prepare("CREATE UNIQUE INDEX IF NOT EXISTS time_hash ON processed (time, hash)")
	if err != nil {
		log.Error(err.Error())
		panic(err.Error())
	}
	stmt.Exec()
}

func importAll() {
	dbInit()
	for _, file := range getFilesToProcess() {
		err := processFile(file)
		if err != nil {
			log.WithField("file", file).Error(err.Error())
		}
	}
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "importgoblin",
		Short: "Image rename and move tool",
	}

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "process all files for import",
		Run: func(cmd *cobra.Command, args []string) {
			importAll()
		},
	}

	importCmd.Flags().StringVarP(&ic.from, "from", "f", "", "Location to import from")
	importCmd.Flags().StringVarP(&ic.to, "to", "t", "", "Location to put imported files to")
	usr, _ := user.Current()
	homeDir := usr.HomeDir
	importCmd.Flags().StringVarP(&ic.db, "db", "d", homeDir+"/.importgoblin/importgoblin.sqlite3", "Location of the database")
	importCmd.Flags().BoolVar(&ic.delete, "delete", false, "If set, original files will be deleted as part of the import process")
	importCmd.Flags().BoolVarP(&ic.force, "force-import", "i", false, "force import even if already processed")

	rootCmd.AddCommand(importCmd)

	rootCmd.Execute()
}
