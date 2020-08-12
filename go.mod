module github.com/matt-doug-davidson/sa-report-querier

go 1.14

replace github.com/matt-doug-davidson/samqttif => /home/allied/go/src/github.com/matt-doug-davidson/samqttif

replace github.com/matt-doug-davidson/saswaggerif => /home/allied/go/src/github.com/matt-doug-davidson/saswaggerif

replace github.com/matt-doug-davidson/fifo => /home/allied/go/src/github.com/matt-doug-davidson/fifo
replace github.com/matt-doug-davidson/timestamps => /home/allied/go/src/github.com/matt-doug-davidson/timestamps

require (
	github.com/eclipse/paho.mqtt.golang v1.2.0 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/matt-doug-davidson/fifo v0.0.0
	github.com/matt-doug-davidson/minutemarktimer v0.0.3
	github.com/matt-doug-davidson/samqttif v0.0.0
	github.com/matt-doug-davidson/saswaggerif v0.0.0
	github.com/matt-doug-davidson/timestamps v0.0.2
	github.com/project-flogo/core v1.0.0 // indirect
	gopkg.in/yaml.v2 v2.3.0
)
