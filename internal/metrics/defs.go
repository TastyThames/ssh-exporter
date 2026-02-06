package metrics

const (
	// exporter health
	MetricExporterUp = "ssh_exporter_up"

	// target health
	MetricTargetUp        = "ssh_target_up"
	MetricScrapeDuration  = "ssh_target_scrape_duration_seconds"
	MetricLastScrapeTs    = "ssh_target_last_scrape_timestamp_seconds"
	MetricCacheAgeSeconds = "ssh_exporter_scrape_cache_seconds"

	// optional
	MetricTargetError = "ssh_target_error"
)
