package exporter

import (
	"log"
	"time"

	"github.com/bah2830/unifi_exporter/pkg/unifi/api"
	"github.com/prometheus/client_golang/prometheus"
)

// A DeviceCollector is a Prometheus collector for metrics regarding Ubiquiti
// UniFi devices.
type DeviceCollector struct {
	Devices          *prometheus.Desc
	AdoptedDevices   *prometheus.Desc
	UnadoptedDevices *prometheus.Desc

	UptimeSecondsTotal *prometheus.Desc

	ReceivedBytesTotal      *prometheus.Desc
	TransmittedBytesTotal   *prometheus.Desc
	ReceivedPacketsTotal    *prometheus.Desc
	TransmittedPacketsTotal *prometheus.Desc
	TransmittedDroppedTotal *prometheus.Desc

	Stations *prometheus.Desc

	c     *api.Client
	sites []*api.Site
}

// Verify that the Exporter implements the collector interface.
var _ collector = &DeviceCollector{}

// NewDeviceCollector creates a new DeviceCollector which collects metrics for
// a specified site.
func NewDeviceCollector(c *api.Client, sites []*api.Site) *DeviceCollector {
	const (
		subsystem = "devices"
	)

	var (
		labelsSiteOnly       = []string{"site"}
		labelsUptime         = []string{"site", "id", "mac", "name"}
		labelsDevice         = []string{"site", "id", "mac", "name", "connection"}
		labelsDeviceStations = []string{"site", "id", "mac", "name", "interface", "radio", "user_type"}
	)

	return &DeviceCollector{
		Devices: prometheus.NewDesc(
			// Subsystem is used as name so we get "unifi_devices"
			prometheus.BuildFQName(namespace, "", subsystem),
			"Total number of devices",
			labelsSiteOnly,
			nil,
		),

		AdoptedDevices: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "adopted"),
			"Number of devices which are adopted",
			labelsSiteOnly,
			nil,
		),

		UnadoptedDevices: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "unadopted"),
			"Number of devices which are not adopted",
			labelsSiteOnly,
			nil,
		),

		UptimeSecondsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "uptime_seconds_total"),
			"Device uptime in seconds",
			labelsUptime,
			nil,
		),

		ReceivedBytesTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "received_bytes_total"),
			"Number of bytes received by devices",
			labelsDevice,
			nil,
		),

		TransmittedBytesTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "transmitted_bytes_total"),
			"Number of bytes transmitted by devices",
			labelsDevice,
			nil,
		),

		ReceivedPacketsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "received_packets_total"),
			"Number of packets received by devices",
			labelsDevice,
			nil,
		),

		TransmittedPacketsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "transmitted_packets_total"),
			"Number of packets transmitted by devices",
			labelsDevice,
			nil,
		),

		TransmittedDroppedTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "transmitted_packets_dropped_total"),
			"Number of packets which are dropped on transmission by devices",
			labelsDevice,
			nil,
		),

		Stations: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "stations"),
			"Total number of stations (clients) connected to devices",
			labelsDeviceStations,
			nil,
		),

		c:     c,
		sites: sites,
	}
}

// collect begins a metrics collection task for all metrics related to UniFi
// devices.
func (c *DeviceCollector) collect(ch chan<- prometheus.Metric) (*prometheus.Desc, error) {
	for _, s := range c.sites {
		devices, err := c.c.Devices(s.Name)
		if err != nil {
			return c.Devices, err
		}

		ch <- prometheus.MustNewConstMetric(
			c.Devices,
			prometheus.GaugeValue,
			float64(len(devices)),
			s.Description,
		)

		c.collectDeviceAdoptions(ch, s.Description, devices)
		c.collectDeviceUptime(ch, s.Description, devices)
		c.collectDeviceBytes(ch, s.Description, devices)
		c.collectDeviceStations(ch, s.Description, devices)
	}

	return nil, nil
}

// collectDeviceAdoptions collects counts for number of adopted and unadopted
// UniFi devices.
func (c *DeviceCollector) collectDeviceAdoptions(ch chan<- prometheus.Metric, siteLabel string, devices []*api.Device) {
	var adopted, unadopted int

	for _, d := range devices {
		if d.Adopted {
			adopted++
		} else {
			unadopted++
		}
	}

	ch <- prometheus.MustNewConstMetric(
		c.AdoptedDevices,
		prometheus.GaugeValue,
		float64(adopted),
		siteLabel,
	)

	ch <- prometheus.MustNewConstMetric(
		c.UnadoptedDevices,
		prometheus.GaugeValue,
		float64(unadopted),
		siteLabel,
	)
}

// collectDeviceUptime collects device uptime for UniFi devices.
func (c *DeviceCollector) collectDeviceUptime(ch chan<- prometheus.Metric, siteLabel string, devices []*api.Device) {
	for _, d := range devices {
		labels := []string{
			siteLabel,
			d.ID,
			d.NICs[0].MAC.String(),
			d.Name,
		}

		ch <- prometheus.MustNewConstMetric(
			c.UptimeSecondsTotal,
			prometheus.CounterValue,
			float64(d.Uptime/time.Second),
			labels...,
		)
	}
}

// collectDeviceBytes collects receive and transmit byte counts for UniFi devices.
func (c *DeviceCollector) collectDeviceBytes(ch chan<- prometheus.Metric, siteLabel string, devices []*api.Device) {
	for _, d := range devices {
		labels := []string{
			siteLabel,
			d.ID,
			d.NICs[0].MAC.String(),
			d.Name,
		}

		ch <- prometheus.MustNewConstMetric(
			c.ReceivedBytesTotal,
			prometheus.CounterValue,
			float64(d.Stats.All.ReceiveBytes),
			append(labels, "user")...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TransmittedBytesTotal,
			prometheus.CounterValue,
			float64(d.Stats.All.TransmitBytes),
			append(labels, "user")...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.ReceivedPacketsTotal,
			prometheus.CounterValue,
			float64(d.Stats.All.ReceivePackets),
			append(labels, "user")...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TransmittedPacketsTotal,
			prometheus.CounterValue,
			float64(d.Stats.All.TransmitPackets),
			append(labels, "user")...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TransmittedDroppedTotal,
			prometheus.CounterValue,
			float64(d.Stats.All.TransmitDropped),
			append(labels, "user")...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.ReceivedBytesTotal,
			prometheus.CounterValue,
			float64(d.Stats.Uplink.ReceiveBytes),
			append(labels, "uplink")...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TransmittedBytesTotal,
			prometheus.CounterValue,
			float64(d.Stats.Uplink.TransmitBytes),
			append(labels, "uplink")...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.ReceivedPacketsTotal,
			prometheus.CounterValue,
			float64(d.Stats.Uplink.ReceivePackets),
			append(labels, "uplink")...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TransmittedPacketsTotal,
			prometheus.CounterValue,
			float64(d.Stats.Uplink.TransmitPackets),
			append(labels, "uplink")...,
		)
	}
}

// collectDeviceStations collects station counts for UniFi devices.
func (c *DeviceCollector) collectDeviceStations(ch chan<- prometheus.Metric, siteLabel string, devices []*api.Device) {
	for _, d := range devices {
		labels := []string{
			siteLabel,
			d.ID,
			d.NICs[0].MAC.String(),
			d.Name,
		}

		for _, r := range d.Radios {
			// Since the radio name and type will be different for each
			// radio, we copy the original labels slice and append, to avoid
			// mutating it
			llabels := make([]string, len(labels))
			copy(llabels, labels)
			llabels = append(llabels, r.Name, r.Radio)

			ch <- prometheus.MustNewConstMetric(
				c.Stations,
				prometheus.GaugeValue,
				float64(r.Stats.NumberUserStations),
				append(llabels, "private")...,
			)
			ch <- prometheus.MustNewConstMetric(
				c.Stations,
				prometheus.GaugeValue,
				float64(r.Stats.NumberGuestStations),
				append(llabels, "guest")...,
			)
		}
	}
}

// Describe sends the descriptors of each metric over to the provided channel.
// The corresponding metric values are sent separately.
func (c *DeviceCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		c.Devices,
		c.AdoptedDevices,
		c.UnadoptedDevices,

		c.UptimeSecondsTotal,

		c.ReceivedBytesTotal,
		c.TransmittedBytesTotal,
		c.ReceivedPacketsTotal,
		c.TransmittedPacketsTotal,
		c.TransmittedDroppedTotal,

		c.Stations,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect is the same as CollectError, but ignores any errors which occur.
// Collect exists to satisfy the prometheus.Collector interface.
func (c *DeviceCollector) Collect(ch chan<- prometheus.Metric) {
	_ = c.CollectError(ch)
}

// CollectError sends the metric values for each metric pertaining to the global
// cluster usage over to the provided prometheus Metric channel, returning any
// errors which occur.
func (c *DeviceCollector) CollectError(ch chan<- prometheus.Metric) error {
	if desc, err := c.collect(ch); err != nil {
		ch <- prometheus.NewInvalidMetric(desc, err)
		log.Printf("[ERROR] failed collecting device metric %v: %v", desc, err)
		return err
	}

	return nil
}
