package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/matt-doug-davidson/minutemarktimer"
	"github.com/matt-doug-davidson/samqttif"
	"github.com/matt-doug-davidson/saswaggerif"
	"github.com/matt-doug-davidson/timestamps"
	"gopkg.in/yaml.v2"
)

var vehicleClassReportedList = []string{"Car", "Motorbike", "Bus", "Truck", "Van", "Pickup", "Unknown"}

type Config struct {
	SA struct {
		Host            string `yaml:"host"`
		SwaggerPort     string `yaml:"swagger-port"`
		SwaggerUsername string `yaml:"swagger-username"`
		SwaggerPassword string `yaml:"swagger-password"`
		MQTTHost        string `yaml:"mqtt-host"`
		MQTTPort        string `yaml:"mqtt-port"`
		MQTTClientID    string `yaml:"mqtt-client-id"`
	} `yaml:"sa"`
	Report struct {
		Endpoints []struct {
			Ingress string `yaml:"ingress"`
			Egress  string `yaml:"egress"`
		} `yaml:"endpoints"`
		IngressChunkInterval int64 `yaml:"ingress-chunk-interval"`
		IngressChunks        int64 `yaml:"ingress-chunks"`
	} `yaml:"report"`
	Debug bool `yaml:"debug"`
}

type ClassEgress struct {
	Total               int64
	TransitCount        int64
	TransitAccumulation int64
}

// AggregateEgress holds the statistical information for a specific egress path.
type AggregateEgress struct {
	Total               int64
	TransitCount        int64
	TransitAccumulation int64
	Class               map[string]ClassEgress
}

// AggregateIngress hols the statistical information for a specific ingress path.
type AggregateIngress struct {
	Total int64
	Class map[string]int64
}

type TrackingData struct {
	Timestamp int64
	Class     string
}
type SAReportQueryConnector struct {
	Conf           *Config
	SAMqttClient   *samqttif.SAMqttClient
	SASwagger      *saswaggerif.SASwaggerIF
	IChunkCount    int64
	LatestChunk    int64
	Ingress        IngressChunks
	IngressStats   map[string]AggregateIngress
	EgressStats    map[string]AggregateEgress
	ReportEntities []string
	Debug          bool
}
type NumberTimestampMap map[string]TrackingData

// IngressChunk contains tracking data for a chunk interval
// Key is the licence plath number string.
type IngressChunk map[string]TrackingData

// IngressChunks contains tracking data for each ingress entity path.
// Key is entity path string.
type IngressChunks map[string][]IngressChunk

func NewSAReportQueryConnector(config *Config, sa *samqttif.SAMqttClient, sw *saswaggerif.SASwaggerIF) *SAReportQueryConnector {
	sarqc := &SAReportQueryConnector{Conf: config, SAMqttClient: sa, SASwagger: sw}
	sarqc.IChunkCount = config.Report.IngressChunks
	sarqc.LatestChunk = sarqc.IChunkCount - 1

	sarqc.Ingress = make(IngressChunks)
	sarqc.IngressStats = make(map[string]AggregateIngress)
	sarqc.EgressStats = make(map[string]AggregateEgress)
	// Each ingress will have a list of chunks
	for _, v := range config.Report.Endpoints {
		sarqc.Ingress[v.Ingress] = make([]IngressChunk, 0)
		for ic := int64(0); ic < sarqc.IChunkCount; ic++ {
			sarqc.Ingress[v.Ingress] = append(sarqc.Ingress[v.Ingress], make(IngressChunk))
		}
		sarqc.ReportEntities = append(sarqc.ReportEntities, v.Egress)
		// Initialize ingress statistics
		ai := AggregateIngress{0, make(map[string]int64)}
		for _, vc := range vehicleClassReportedList {
			ai.Class[vc] = 0
		}
		sarqc.IngressStats[v.Ingress] = ai

		// Updating stats examples
		sarqc.IngressStats[v.Ingress].Class["Car"] = 0
		ist := sarqc.IngressStats[v.Ingress]
		ist.Total = 0
		sarqc.IngressStats[v.Ingress] = ist
		is := sarqc.IngressStats[v.Ingress].Class["Car"]
		is = 0
		sarqc.IngressStats[v.Ingress].Class["Car"] = is

		// Initialize egress statistics
		ei := AggregateEgress{0, 0, 0, make(map[string]ClassEgress)}
		for _, vc := range vehicleClassReportedList {
			ce := ClassEgress{0, 0, 0}
			ei.Class[vc] = ce
		}
		sarqc.EgressStats[v.Egress] = ei
	}

	sarqc.Debug = config.Debug

	// onConnect defines the on connect handler which resets backoff variables.
	var onConnectCallback samqttif.MqttCallback = func() {
		fmt.Println("SA Report Querier connected to MQTT broker")
		sarqc.setEntityPathsRunning()
	}

	// onDisconnect defines the connection lost handler for the mqtt client.
	var onDisconnectCallback samqttif.MqttCallback = func() {
		fmt.Println("SA Report Querier disconnected from MQTT broker")
	}
	sarqc.SAMqttClient.RegisterConnectionCallbacks(onConnectCallback, onDisconnectCallback)

	return sarqc
}

func (sarqc *SAReportQueryConnector) setEntityPathsRunning() {
	// for _, entity := range sarqc.ReportEntities {
	//	sarqc.SAMqttClient.PublishRunning(entity)
	// }
}

func (sarqc *SAReportQueryConnector) PrepareForNewValues() {
	for _, v := range sarqc.Conf.Report.Endpoints {
		ig := sarqc.Ingress[v.Ingress]
		for ic := int64(1); ic < sarqc.IChunkCount; ic++ {
			ig[ic-1] = nil
			ig[ic-1] = ig[ic]
		}
		ig[sarqc.LatestChunk] = make(IngressChunk)
	}
}

func (sarqc *SAReportQueryConnector) EnableCleanup() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		// This connector does not control state of the entity paths
		sarqc.SAMqttClient.Cleanup()
		// Do clean-up activities here such as:

		// - Anything else?
		os.Exit(1)
	}()
}

func (sarqc *SAReportQueryConnector) PrintIngress(ingress string) {
	fmt.Println("Ingress Maps for ", ingress)
	for i, v := range sarqc.Ingress[ingress] {
		//fmt.Println(i, v)
		fmt.Println("Map ", i)
		for k, d := range v {
			fmt.Println(k, d)
		}
	}

}

func (sarqc *SAReportQueryConnector) clearStats() {
	for _, v := range sarqc.Conf.Report.Endpoints {

		ism := sarqc.IngressStats[v.Ingress]
		ism.Total = 0
		for _, vc := range vehicleClassReportedList {
			ism.Class[vc] = 0
		}
		sarqc.IngressStats[v.Ingress] = ism

		esm := sarqc.EgressStats[v.Egress]
		esm.Total = 0
		esm.TransitCount = 0
		esm.TransitAccumulation = 0
		for _, vc := range vehicleClassReportedList {
			esmc := esm.Class[vc]
			esmc.Total = 0
			esmc.TransitCount = 0
			esmc.TransitAccumulation = 0
			esm.Class[vc] = esmc
		}
		sarqc.EgressStats[v.Egress] = esm
	}
}

func averageSeconds(accumulator int64, number int64) float64 {
	if number == 0 {
		return 0.0
	}
	return (float64(accumulator) / float64(number)) / 1000000000
}

func (sarqc *SAReportQueryConnector) reportStats() {
	datetime := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	for _, v := range sarqc.Conf.Report.Endpoints {
		svm := samqttif.NewSensorValueMessage(v.Egress, datetime)

		svm.AddValue("TotalTransits", float64(sarqc.EgressStats[v.Egress].TransitCount))
		svm.AddValue("TotalTransitAccumulation", float64(sarqc.EgressStats[v.Egress].TransitAccumulation))
		averagedSeconds := 0.0
		if sarqc.EgressStats[v.Egress].TransitCount > 0 {
			averagedSeconds = averageSeconds(sarqc.EgressStats[v.Egress].TransitAccumulation, sarqc.EgressStats[v.Egress].TransitCount)
		}
		svm.AddValue("TotalAverage", averagedSeconds)
		for _, vc := range vehicleClassReportedList {
			evc := sarqc.EgressStats[v.Egress].Class[vc]
			svm.AddValue(vc+"Transits", float64(evc.TransitCount))
			svm.AddValue(vc+"TransitAccumulation", float64(evc.TransitAccumulation))
			averagedSeconds := averageSeconds(evc.TransitAccumulation, evc.TransitCount)
			svm.AddValue(vc+"TransitAverage", averagedSeconds)
		}
		if svm.AnyValues() {
			sarqc.SAMqttClient.PublishValueMessage(svm)
		} else {
			fmt.Println("SVM contains no values")
			fmt.Println(svm)
		}
	}

}

func (sarqc *SAReportQueryConnector) process() {
	now := time.Now()
	endTimestamp := timestamps.RoundDownMinutes(now.UnixNano())
	//end := timestamps.TimestampToUTCZTimestring(endTimestamp)

	startTimestamp := timestamps.SubtractMinutes(endTimestamp, sarqc.Conf.Report.IngressChunkInterval)
	start := timestamps.TimestampToUTCZTimestring(startTimestamp)

	endTimestamp -= 1000000
	end := timestamps.TimestampToUTCZTimestring(endTimestamp)

	// Round now to nearest minute (00.000 seconds). This will be the end timestamp
	// Add the chunk interval to get the start time.
	//
	topOfHour := timestamps.IsTimeFirstMinuteOfHour(now)

	sarqc.PrepareForNewValues()

	for _, v := range sarqc.Conf.Report.Endpoints {
		ingressEntities := []string{v.Ingress}
		rc, values := sarqc.SASwagger.FindValuesForEntity(ingressEntities, start, end)
		if rc != 200 {
			break
		}

		ingressStats := sarqc.IngressStats[v.Ingress]
		im := sarqc.Ingress[v.Ingress][sarqc.LatestChunk]
		for _, vv := range values.Output {
			for _, vvv := range vv.Values {
				ingressStats.Total++
				a := vvv.Attributes
				timestamp := vv.Datetime.UnixNano()
				if sarqc.Debug {
					fmt.Println(a.Number, a.VehicleClass, a.Probability, vv.Datetime, timestamp)
				}

				im[a.Number] = TrackingData{Class: a.VehicleClass, Timestamp: timestamp}
				ingressStats.Class[a.VehicleClass]++
			}
		}
		sarqc.IngressStats[v.Ingress] = ingressStats

		egressEntities := []string{v.Egress}
		rc, values = sarqc.SASwagger.FindValuesForEntity(egressEntities, start, end)
		if rc != 200 {
			break
		}

		egressStats := sarqc.EgressStats[v.Egress]
		for _, vv := range values.Output {
			for _, vvv := range vv.Values {
				egressStats.Total++

				a := vvv.Attributes
				vcs := egressStats.Class[a.VehicleClass]
				vcs.Total++
				timestamp := vv.Datetime.UnixNano()
				if sarqc.Debug {
					fmt.Println(a.Number, a.VehicleClass, a.Probability, vv.Datetime, timestamp)
				}
				for i := sarqc.LatestChunk; i >= 0; i-- {
					x, found := sarqc.Ingress[v.Ingress][i][a.Number]
					if found {
						diff := timestamp - x.Timestamp
						if a.VehicleClass != "Unknown" {
							egressStats.TransitCount++
							egressStats.TransitAccumulation += diff
							vcs.TransitCount++
							vcs.TransitAccumulation += diff
						}
						if sarqc.Debug {
							fmt.Println("#### Found ", i, a.Number, x)
							fmt.Println("Diff: ", diff)
						}
						break
					}
				}
				egressStats.Class[a.VehicleClass] = vcs // Re-apply updated Vehicle class stats
			}
		}
		sarqc.EgressStats[v.Egress] = egressStats // Re-apply update stats
	}

	///sarqc.reportStats()
	if topOfHour {
		//if true {
		// Report for the egress.
		sarqc.reportStats()
		// Clear the statisics associated with the egress and ingress.
		sarqc.clearStats()
	}

}

func (sarqc *SAReportQueryConnector) start() {
	var oneHourCallback minutemarktimer.MinuteMarkTimerCallback = func() {
		sarqc.process()
	}
	mmt := minutemarktimer.NewMinuteMarkTimer()
	mmt.AddTimer(sarqc.Conf.Report.IngressChunkInterval, 0, oneHourCallback)
	mmt.Start()
}

func isRunningInDockerContainer() bool {
	// docker creates a .dockerenv file at the root
	// of the directory tree inside the container.
	// if this file exists then the viewer is running
	// from inside a container so return true

	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	return false
}

func main() {

	var configFile string
	if isRunningInDockerContainer() {
		// Container path
		configFile = "/data/" + os.Getenv("CONFIG")
	} else {
		// Absolute path
		configFile = os.Getenv("CONFIG")
	}

	// Create config structure
	config := &Config{}

	// Open config file
	file, err := os.Open(configFile)
	if err != nil {
		panicMsg := fmt.Sprintf("Error in opening configuration file, %s. Cause: %s\n", configFile, err.Error())
		panic(panicMsg)
	}
	defer file.Close()
	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		fmt.Println("Error in decode YAML from file. Cause: ", err.Error())
	}

	mqttHost := config.SA.Host
	if config.SA.MQTTHost != "" {
		mqttHost = config.SA.MQTTHost
	}

	esp := false
	saClient := samqttif.NewSAMqttClient(
		mqttHost,
		config.SA.MQTTPort,
		config.SA.MQTTClientID,
		esp, // 4 = false, 5 = true,
		config.Debug)

	saSwagIf := saswaggerif.NewSASwaggerIF(config.SA.Host, config.SA.SwaggerPort, config.SA.SwaggerUsername,
		config.SA.SwaggerPassword, config.Debug)

	connector := NewSAReportQueryConnector(config, saClient, saSwagIf)

	connector.EnableCleanup()

	saClient.Connect()

	connector.start()

	for {
		time.Sleep(1 * time.Second)
	}

}
