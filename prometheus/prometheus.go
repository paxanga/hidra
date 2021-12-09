package prometheus

import (
	"log"
	"net/http"
	"time"

	"github.com/hidracloud/hidra/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsToPrint struct {
	Labels []string
	Value  float64
	Name   string
}

// StartPrometheus starts the prometheus server
func StartPrometheus(listenAddr string, pullTime int) {
	go func() {
		gaugeDict := make(map[string]*prometheus.GaugeVec)
		labelSet := make(map[string][]string, 0)

		for {
			samples, err := models.GetSamples()

			metricsToPrint := make([]MetricsToPrint, 0)
			if err != nil {
				log.Fatal(err)
				return
			}

			for _, sample := range samples {
				tmp, err := models.GetSampleResultBySampleIDWithLimit(sample.ID.String(), 1)
				if len(tmp) == 0 {
					continue
				}

				if err != nil {
					log.Fatal(err)
					return
				}

				latestSampleResult := tmp[0]

				metrics, err := models.GetMetricsBySampleResultID(latestSampleResult.ID.String())

				if err != nil {
					log.Fatal(err)
				}

				for _, metric := range metrics {
					metricLabels, err := models.GetMetricLabelByMetricID(metric.ID)

					if err != nil {
						log.Fatal(err)
					}

					if _, ok := gaugeDict[metric.Name]; !ok {
						labelsKey := make([]string, len(metricLabels))
						for i, label := range metricLabels {
							labelsKey[i] = label.Key
						}

						gaugeDict[metric.Name] = prometheus.NewGaugeVec(
							prometheus.GaugeOpts{
								Namespace: "hidra",
								Name:      metric.Name,
								Help:      metric.Description,
							},
							labelsKey,
						)

						labelSet[metric.Name] = labelsKey

						prometheus.MustRegister(gaugeDict[metric.Name])
					}

					labelsDict := make(map[string]string)

					for _, label := range metricLabels {
						labelsDict[label.Key] = label.Value
					}

					labels := make([]string, len(labelSet[metric.Name]))
					for k, v := range labelSet[metric.Name] {
						if _, ok := labelsDict[v]; ok {
							labels[k] = labelsDict[v]
						}
					}

					metricsToPrint = append(metricsToPrint, MetricsToPrint{
						Labels: labels,
						Value:  metric.Value,
						Name:   metric.Name,
					})
				}
			}

			// Reset all the metrics
			for _, gauge := range gaugeDict {
				gauge.Reset()
			}

			for _, metric := range metricsToPrint {
				gaugeDict[metric.Name].WithLabelValues(metric.Labels...).Set(metric.Value)
			}

			time.Sleep(time.Second * time.Duration(pullTime))
		}
	}()

	log.Println("Starting metrics at " + listenAddr)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
