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
	fmt.Fprintln(w, "# HELP ssh_exporter_up 1 if exporter process is running.")
	fmt.Fprintln(w, "# TYPE ssh_exporter_up gauge")
	fmt.Fprintf(w, "%s 1\n", MetricExporterUp)

	fmt.Fprintln(w, "# HELP ssh_exporter_render_duration_seconds Time spent rendering /metrics.")
	fmt.Fprintln(w, "# TYPE ssh_exporter_render_duration_seconds gauge")

	// ---------------------------------------------------
	// Target metrics (headers)
	// ---------------------------------------------------
	fmt.Fprintln(w, "# HELP ssh_target_up 1 if last SSH scrape succeeded.")
	fmt.Fprintln(w, "# TYPE ssh_target_up gauge")

	fmt.Fprintln(w, "# HELP ssh_target_scrape_cache_age_seconds Age of cached result per target.")
	fmt.Fprintln(w, "# TYPE ssh_target_scrape_cache_age_seconds gauge")

	fmt.Fprintln(w, "# HELP ssh_target_scrape_duration_seconds Duration of SSH scrape.")
	fmt.Fprintln(w, "# TYPE ssh_target_scrape_duration_seconds gauge")

	fmt.Fprintln(w, "# HELP ssh_target_last_scrape_timestamp_seconds Unix timestamp of last scrape.")
	fmt.Fprintln(w, "# TYPE ssh_target_last_scrape_timestamp_seconds gauge")

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

		labels := map[string]string{
			"target": t,
		}
		for k, v := range res.Labels {
			labels[k] = v
		}

		// cache age
		age := now.Sub(res.At).Seconds()
		fmt.Fprintf(w, "ssh_target_scrape_cache_age_seconds%s %.3f\n",
			formatLabels(labels), age)

		// up metric
		up := 1.0
		if res.Err != nil {
			up = 0
		}
		fmt.Fprintf(w, "%s%s %.0f\n",
			MetricTargetUp, formatLabels(labels), up)

		// extra values from worker
		for name, val := range res.Values {
			if name == MetricTargetUp || name == MetricExporterUp {
				continue
			}
			fmt.Fprintf(w, "%s%s %v\n",
				name, formatLabels(labels), val)
		}
	}

	// render duration
	dur := time.Since(start).Seconds()
	fmt.Fprintf(w, "ssh_exporter_render_duration_seconds %.6f\n", dur)
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
		fmt.Fprintf(&b, `%s="%s"`, k, m[k])
	}
	b.WriteString("}")
	return b.String()
}
