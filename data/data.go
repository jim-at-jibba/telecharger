package data

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type QueueItem struct {
	Id             int
	VideoId        string
	OutputName     string
	EmbedThumbnail bool
	AudioOnly      bool
	AudioFormat    string
	ExtraCommands  string
	Status         string
}

var db *sql.DB

func OpenDatabase() error {
	var err error

	if len(os.Getenv("DEBUG")) > 0 {
		db, err = sql.Open("sqlite3", "./sqlite-database-dev.db")
	} else {
		dirname, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		path := fmt.Sprintf("%s/telecharger", dirname)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			err := os.Mkdir(path, os.ModePerm)
			if err != nil {
				log.Print(err.Error())
			}
		}

		db, err = sql.Open("sqlite3", fmt.Sprintf("%s/sqlite-database.db", path))
		if err != nil {
			log.Print(err.Error())
			return err
		}
	}
	if err != nil {
		log.Print(err.Error())
		return err
	}

	return db.Ping()
}

func CreateQueueTable() {
	createTableSQL := `CREATE TABLE IF NOT EXISTS queue (
		"Id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"VideoId" TEXT NOT NULL,
    "OutputName" TEXT NOT NULL,
		"EmbedThumbnail" BOOL NOT NULL,
		"AudioOnly" BOOL NOT NULL,
    "AudioFormat" TEXT NOT NULL,
    "Status" TEXT NOT NULL,
    "ExtraCommands" TEXT NOT NULL
	  );`

	statement, err := db.Prepare(createTableSQL)
	if err != nil {
		log.Fatal(err.Error())
	}

	_, err = statement.Exec()
	if err != nil {
		log.Fatalln(err)
	}
}

func InsertQueueItem(videoId, outputName, audioFormat, extraCommnds string, embedThumbnail, audioOnly bool) error {
	insertNoteSQL := `INSERT INTO queue(videoId, outputName, audioFormat, extraCommands, embedThumbnail, audioOnly, status) VALUES (?, ?, ?, ?, ?, ?, ?)`
	statement, err := db.Prepare(insertNoteSQL)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = statement.Exec(videoId, outputName, audioFormat, extraCommnds, embedThumbnail, audioOnly, "queued")
	if err != nil {
		log.Fatalln(err)
		return err
	}

	return nil
}

func UpdateQueueItemStatus(id int, status string) error {
	insertNoteSQL := `UPDATE queue SET Status = ? WHERE id = ?`
	statement, err := db.Prepare(insertNoteSQL)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = statement.Exec(status, id)
	if err != nil {
		log.Fatalln(err)
		return err
	}

	return nil
}

func DeleteQueueItem(id int) error {
	deleteItemSQL := `DELETE FROM queue WHERE id = ?`
	statement, err := db.Prepare(deleteItemSQL)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = statement.Exec(id)
	if err != nil {
		log.Fatalln(err)
		return err
	}

	return nil
}

func GetAllQueueItems(status string) ([]*QueueItem, error) {
	row, err := db.Query("SELECT * FROM queue WHERE Status = $1", status)
	if err != nil {
		log.Fatal(err)
	}

	defer row.Close()

	queueItems := []*QueueItem{}

	for row.Next() {
		var queueItem QueueItem

		row.Scan(
			&queueItem.Id,
			&queueItem.VideoId,
			&queueItem.OutputName,
			&queueItem.EmbedThumbnail,
			&queueItem.AudioOnly,
			&queueItem.AudioFormat,
			&queueItem.Status,
			&queueItem.ExtraCommands,
		)

		queueItems = append(queueItems, &queueItem)
	}

	return queueItems, nil
}
