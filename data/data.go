package data

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type QueueItem struct {
	Id             int
	VideoId        string
	OutputName     string
	EmbedThumbnail bool
	AudioOnly      bool
	AudioFormat    string
	Status         string
}

var db *sql.DB

func OpenDatabase() error {
	var err error

	db, err = sql.Open("sqlite3", "./sqlite-database.db")
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
    "Status" TEXT NOT NULL
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

func InsertQueueItem(word, definition, category string) {
	insertNoteSQL := `INSERT INTO studybuddy(word, definition, category) VALUES (?, ?, ?)`
	statement, err := db.Prepare(insertNoteSQL)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = statement.Exec(word, definition, category)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Inserted study note successfully")
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
		)

		queueItems = append(queueItems, &queueItem)
	}

	return queueItems, nil
}
