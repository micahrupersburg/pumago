package content

import (
	"os"
	"path/filepath"
)

func SafariBrowser() *Browser {
	historyPath := filepath.Join(os.Getenv("HOME"), "Library/Safari/History.db")
	return &Browser{
		historyPath: historyPath,
		query: `SELECT 
    COALESCE(history_visits.title, '') AS title,
    history_items.url,
    CAST(history_visits.visit_time AS INT) AS visit_time
FROM 
    history_items
INNER JOIN 
    history_visits ON history_items.id = history_visits.history_item
WHERE 
    history_visits.visit_time > ?
    AND history_visits.id IN (
        SELECT MAX(history_visits.id)
        FROM history_visits
        INNER JOIN history_items ON history_visits.history_item = history_items.id
        GROUP BY history_items.url
    )

ORDER BY visit_time DESC 
LIMIT 100;`,
		origin: SAFARI,
	}
}
