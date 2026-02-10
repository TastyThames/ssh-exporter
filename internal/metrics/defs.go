package metrics

const (
	// exporter health
	MetricExporterUp = "ssh_exporter_up"

	// target health
	MetricTargetUp       = "ssh_target_up"
	MetricScrapeDuration = "ssh_target_scrape_duration_seconds"
	MetricLastScrapeTs   = "ssh_target_last_scrape_timestamp_seconds"

	// cache + render
	MetricCacheAgeSeconds       = "ssh_target_scrape_cache_age_seconds"
	MetricRenderDurationSeconds = "ssh_exporter_render_duration_seconds"

	// error flag
	MetricTargetError = "ssh_target_error"
)
