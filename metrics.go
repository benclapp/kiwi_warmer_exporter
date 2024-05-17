package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type DeviceInfo struct {
	Result               string `json:"result"`
	DeviceMAC            string `json:"deviceId"`
	DeviceName           string `json:"deviceName"`
	DefaultTemperature   string `json:"defaultTemperature"`
	CurrentTemperature   string `json:"currentTemperature"`
	CurrentHumidity      string `json:"currentHumidity"`
	OriginalTemperature  string `json:"originalTemperature"`
	CurrentTemperature2  string `json:"currentTemperature2"`
	CurrentHumidity2     string `json:"currentHumidity2"`
	OriginalTemperature2 string `json:"originalTemperature2"`
	IsHeating            string `json:"isHeating"`
	TimeSlotTemperatures string `json:"timeSlotTemperatures"`
	WorkTemperature      string `json:"workTemperature"`
	NextSlotTemperature  string `json:"nextSlotTemperature"`
	Lat                  string `json:"lat"`
	Lon                  string `json:"lon"`
	Hw                   string `json:"hw"`
	Sw                   string `json:"sw"`
	Error                string `json:"error"`
	HeapHealth           string `json:"heapHealth"`
	DeviceStatus         string `json:"deviceStatus"`
}

var (
	up         = prometheus.NewDesc("kiwi_warmer_up", "Whether the Kiwi Warmer unit responded to the scrape", nil, nil)
	duration   = prometheus.NewDesc("kiwi_warmer_scrape_duration_seconds", "Duration of the query to collect device info", nil, nil)
	info       = prometheus.NewDesc("kiwi_warmer_info", "Info about the Kiwi Warmer device", []string{"MAC_address", "name", "hardware_version", "software_version"}, nil)
	heapHealth = prometheus.NewDesc("kiwi_warmer_heap_health", "Health of the device's heap", nil, nil)
	heating    = prometheus.NewDesc("kiwi_warmer_heating", "Whether the switch is currently heating", nil, nil)
	humidity   = prometheus.NewDesc("kiwi_warmer_humidity_percent", "Current humidity, 0-1", []string{"sensor"}, nil)
	status     = prometheus.NewDesc("kiwi_warmer_device_status", "Device status", nil, nil)
	targetTemp = prometheus.NewDesc("kiwi_warmer_target_temperature_celcius", "Target temperature", nil, nil)
	temp       = prometheus.NewDesc("kiwi_warmer_temperature_celcius", "Current temperature", []string{"sensor"}, nil)
)

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
	ch <- duration
	ch <- info
	ch <- temp
	ch <- humidity
	ch <- heating
	ch <- targetTemp
	ch <- heapHealth
	ch <- status
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {

	di, t, err := getDeviceInfo()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0)
		ch <- prometheus.MustNewConstMetric(duration, prometheus.GaugeValue, t)
		return
	} else {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 1)
		ch <- prometheus.MustNewConstMetric(duration, prometheus.GaugeValue, t)
	}

	ch <- prometheus.MustNewConstMetric(info, prometheus.GaugeValue, 1,
		di.DeviceMAC, di.DeviceName, di.Hw, di.Sw)

	parseAndSetValues("di.CurrentTemperature", di.CurrentTemperature, 100, temp, ch, "0")
	parseAndSetValues("di.CurrentTemperature2", di.CurrentTemperature2, 100, temp, ch, "1")
	parseAndSetValues("di.CurrentHumidity", di.CurrentHumidity, 10000, humidity, ch, "0")
	parseAndSetValues("di.CurrentHumidity2", di.CurrentHumidity2, 10000, humidity, ch, "1")
	parseAndSetValues("di.WorkTemperature", di.WorkTemperature, 1, targetTemp, ch)
	parseAndSetValues("di.HeapHealth", di.HeapHealth, 1, heapHealth, ch)
	parseAndSetValues("di.IsHeating", di.IsHeating, 1, heating, ch)
	parseAndSetValues("di.DeviceStatus", di.DeviceStatus, 1, status, ch)

}

func getDeviceInfo() (DeviceInfo, float64, error) {
	client := http.Client{Timeout: *kwScrapeTimeout}

	start := time.Now()
	resp, err := client.Get(fmt.Sprintf("http://%s/deviceInfo", *kwAddress))

	if err != nil {
		slog.Error("Error querying device", "err", err)
		return DeviceInfo{}, time.Since(start).Seconds(), err
	} else if resp.StatusCode != http.StatusOK {
		slog.Error("Error querying device", "err", err, "status_code", resp.StatusCode)
		return DeviceInfo{}, time.Since(start).Seconds(), err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Error reading response body", "err", err)
		return DeviceInfo{}, time.Since(start).Seconds(), err
	}

	decoded, err := base64.StdEncoding.DecodeString(string(body[:]))
	if err != nil {
		fmt.Println(string(body))
		slog.Error("Error decoding base64 response", "err", err, "body", string(body[:]))
		return DeviceInfo{}, time.Since(start).Seconds(), err
	}

	var di DeviceInfo
	err = json.Unmarshal([]byte(decoded), &di)
	if err != nil {
		slog.Error("Error unmarshalling device info", "err", err)
		return DeviceInfo{}, time.Since(start).Seconds(), err
	}

	return di, time.Since(start).Seconds(), nil
}

func parseAndSetValues(key, value string, div float64, desc *prometheus.Desc, ch chan<- prometheus.Metric, labelValues ...string) {
	slog.Debug("Parsing...",
		"key", key,
		"value", value,
		"div", div,
		"desc", desc,
	)
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		slog.Error("Error parsing device info", key, value)
	} else {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v/div, labelValues...)
	}
}
