package metrics

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/tastythames/ssh-exporter/internal/cache"
)

type Renderer struct {
	Cache cache.Cache
}

func NewRenderer(c cache.Cache) *Renderer {
	return &Renderer{Cache: c}
}

func (r *Renderer) Write(w io.Writer) {
	start := time.Now()
	now := time.Now()

	// ---------------------------------------------------
	// Exporter-level metrics
	// ---------------------------------------------------
	fmt.Fprintf(w, "# HELP %s 1 if exporter process is running.\n", MetricExporterUp)
	fmt.Fprintf(w, "# TYPE %s gauge\n", MetricExporterUp)
	fmt.Fprintf(w, "%s 1\n", MetricExporterUp)

	fmt.Fprintf(w, "# HELP %s Time spent rendering /metrics.\n", MetricRenderDurationSeconds)
	fmt.Fprintf(w, "# TYPE %s gauge\n", MetricRenderDurationSeconds)

	// ---------------------------------------------------
	// Target metrics (headers)
	// ---------------------------------------------------
	fmt.Fprintf(w, "# HELP %s 1 if last SSH scrape succeeded.\n", MetricTargetUp)
	fmt.Fprintf(w, "# TYPE %s gauge\n", MetricTargetUp)

	fmt.Fprintf(w, "# HELP %s Age of cached result per target.\n", MetricCacheAgeSeconds)
	fmt.Fprintf(w, "# TYPE %s gauge\n", MetricCacheAgeSeconds)

	fmt.Fprintf(w, "# HELP %s Duration of SSH scrape.\n", MetricScrapeDuration)
	fmt.Fprintf(w, "# TYPE %s gauge\n", MetricScrapeDuration)

	fmt.Fprintf(w, "# HELP %s Unix timestamp of last scrape.\n", MetricLastScrapeTs)
	fmt.Fprintf(w, "# TYPE %s gauge\n", MetricLastScrapeTs)

	fmt.Fprintf(w, "# HELP %s 1 if last scrape returned error.\n", MetricTargetError)
	fmt.Fprintf(w, "# TYPE %s gauge\n", MetricTargetError)

	// ---------------------------------------------------
	// Snapshot cache
	// ---------------------------------------------------
	snap := r.Cache.Snapshot()

	targets := make([]string, 0, len(snap))
	for t := range snap {
		targets = append(targets, t)
	}
	sort.Strings(targets)

	for _, t := range targets {
		res := snap[t]

		labels := map[string]string{"target": t}
		for k, v := range res.Labels {
			labels[k] = v
		}

		// cache age
		age := now.Sub(res.At).Seconds()
		fmt.Fprintf(w, "%s%s %.3f\n", MetricCacheAgeSeconds, formatLabels(labels), age)

		// up + error
		up := 1.0
		errFlag := 0.0
		if res.Err != nil {
			up = 0
			errFlag = 1
		}
		fmt.Fprintf(w, "%s%s %.0f\n", MetricTargetUp, formatLabels(labels), up)
		fmt.Fprintf(w, "%s%s %.0f\n", MetricTargetError, formatLabels(labels), errFlag)

		// extra values from worker
		for name, val := range res.Values {
			// avoid duplicates + avoid exporter-level metrics
			if name == MetricTargetUp || name == MetricExporterUp || name == MetricTargetError {
				continue
			}
			fmt.Fprintf(w, "%s%s %v\n", name, formatLabels(labels), val)
		}
	}

	// render duration
	dur := time.Since(start).Seconds()
	fmt.Fprintf(w, "%s %.6f\n", MetricRenderDurationSeconds, dur)
}

func formatLabels(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("{")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(",")
		}
		// NOTE: keep simple (labels should not contain quotes in this lab)
		fmt.Fprintf(&b, `%s="%s"`, k, m[k])
	}
	b.WriteString("}")
	return b.String()
}
