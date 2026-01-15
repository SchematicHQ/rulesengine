package rulesengine_test

import (
	"sync"
	"testing"
	"time"

	"github.com/schematichq/rulesengine"
	"github.com/schematichq/rulesengine/null"
	"github.com/stretchr/testify/assert"
)

func TestCompanyAddMetric(t *testing.T) {
	t.Run("adds a new metric when none exists with the same constraints", func(t *testing.T) {
		company := createTestCompany()
		initialLen := len(company.Metrics)

		metric := createTestMetric(company, "test-event", rulesengine.MetricPeriodAllTime, 5)
		company.AddMetric(metric)

		assert.Len(t, company.Metrics, initialLen+1)
		assert.Contains(t, company.Metrics, metric)
	})

	t.Run("replaces existing metric with the same constraints", func(t *testing.T) {
		company := createTestCompany()

		// Add initial metric
		eventSubtype := "test-event"
		period := rulesengine.MetricPeriodAllTime
		monthReset := rulesengine.MetricPeriodMonthResetFirst

		initialMetric := createTestMetric(company, eventSubtype, period, 5)
		company.AddMetric(initialMetric)
		initialLen := len(company.Metrics)

		// Add metric with same constraints but different value
		newMetric := createTestMetric(company, eventSubtype, period, 10)
		company.AddMetric(newMetric)

		// Verify the length hasn't changed but the new metric replaced the old one
		assert.Len(t, company.Metrics, initialLen)
		assert.Contains(t, company.Metrics, newMetric)
		assert.NotContains(t, company.Metrics, initialMetric)

		// Find the metric and verify it has the new value
		foundMetric := company.Metrics.Find(eventSubtype, &period, &monthReset)
		assert.NotNil(t, foundMetric)
		assert.Equal(t, int64(10), foundMetric.Value)
	})

	t.Run("handles concurrent updates safely", func(t *testing.T) {
		company := createTestCompany()

		// Set up concurrent goroutines to add metrics
		const numGoroutines = 10

		// Use a WaitGroup to wait for all goroutines to finish
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				defer wg.Done()

				// Create a metric with a unique event subtype to avoid collision
				uniqueSubtype := "test-event-" + time.Now().String() + "-" + string(rune(index))
				metric := createTestMetric(company, uniqueSubtype, rulesengine.MetricPeriodAllTime, int64(index))

				// Add the metric
				company.AddMetric(metric)
			}(i)
		}

		// Wait for all goroutines to finish
		wg.Wait()

		// Verify that we have at least numGoroutines metrics
		assert.GreaterOrEqual(t, len(company.Metrics), numGoroutines)
	})

	t.Run("nil company does not panic", func(t *testing.T) {
		var company *rulesengine.Company

		// This should not panic
		metric := &rulesengine.CompanyMetric{}
		company.AddMetric(metric)
	})

	t.Run("nil metric does not panic", func(t *testing.T) {
		company := createTestCompany()

		// This should not panic
		company.AddMetric(nil)
	})

	t.Run("company with no metrics does not panic", func(t *testing.T) {
		company := &rulesengine.Company{
			ID:                generateTestID("comp"),
			AccountID:         generateTestID("acct"),
			EnvironmentID:     generateTestID("env"),
			PlanIDs:           []string{generateTestID("plan"), generateTestID("plan")},
			BillingProductIDs: []string{generateTestID("bilp"), generateTestID("bilp")},
			BasePlanID:        null.Nullable(generateTestID("plan")),
			Traits:            make([]*rulesengine.Trait, 0),
			Subscription: &rulesengine.Subscription{
				ID:          generateTestID("bilsub"),
				PeriodStart: time.Now().Add(-30 * 24 * time.Hour),
				PeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
			},
		}

		metric := createTestMetric(company, "foo", rulesengine.MetricPeriodAllTime, int64(1))

		company.AddMetric(metric)
	})
}
