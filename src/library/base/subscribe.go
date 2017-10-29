package base

/**
 [
 	"database"    => $database_name,
    "table"       => $table_name,
    "event"       => [
     	"event_type" => $event_type,
        "time"       => $daytime,
        "data"       => [
            	           "old_data" => []
            	           "new_data" => []
                        ],
                  ],
    "event_index" => $this->event_index
 ];
 */
type subscribe interface {
	OnChange(data map[string] interface{})
}
