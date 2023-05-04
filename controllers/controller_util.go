package controllers

import (
	corev1 "k8s.io/api/core/v1"
	infrav1 "sigs.k8s.io/cluster-api-provider-nested/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"time"
)

const (
	secretErrorRetryDelay = time.Second * 10
	rateLimitWaitTime     = 5 * time.Minute
)

// reconcileRateLimit checks whether a rate limit has been reached and returns whether
// the controller should wait a bit more.
func reconcileRateLimit(setter conditions.Setter) bool {
	condition := conditions.Get(setter, infrav1.RateLimitExceeded)
	if condition != nil && condition.Status == corev1.ConditionTrue {
		if time.Now().Before(condition.LastTransitionTime.Time.Add(rateLimitWaitTime)) {
			// Not yet timed out, reconcileNormal again after timeout
			// Don't give a more precise requeueAfter value to not reconcileNormal too many
			// objects at the same time
			return true
		}
		// Wait time is over, we continue
		conditions.MarkFalse(
			setter,
			infrav1.RateLimitExceeded,
			infrav1.RateLimitNotReachedReason,
			clusterv1.ConditionSeverityInfo,
			"wait time is over. Try reconciling again",
		)
	}
	return false
}
