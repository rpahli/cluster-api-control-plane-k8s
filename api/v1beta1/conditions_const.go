package v1beta1

import clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

const (
	// RateLimitExceeded reports whether the rate limit has been reached.
	RateLimitExceeded clusterv1.ConditionType = "RateLimitExceeded"
	// RateLimitNotReachedReason indicates that the rate limit is not reached yet.
	RateLimitNotReachedReason = "RateLimitNotReached"
)
