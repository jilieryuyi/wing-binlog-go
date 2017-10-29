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
type Subscribe interface {
	Init()
	OnChange(data map[string] interface{})
	Free()
}

const MAX_EVENT_QUEUE_LENGTH = 102400