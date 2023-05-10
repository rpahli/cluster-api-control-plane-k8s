package v1beta1

import clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

const (
	// LoadBalancerAttached reports on whether the load balancer is attached.
	LoadBalancerAttached clusterv1.ConditionType = "LoadBalancerAttached"
	// LoadBalancerUnreachableReason is used when load balancer is unreachable.
	LoadBalancerUnreachableReason = "LoadBalancerUnreachable"
)

const (
	InstanceReadyCondition clusterv1.ConditionType = "InstanceReady"
	PodCreated                                     = "PodCreated"
)

const (
	// RateLimitExceeded reports whether the rate limit has been reached.
	RateLimitExceeded clusterv1.ConditionType = "RateLimitExceeded"
	// RateLimitNotReachedReason indicates that the rate limit is not reached yet.
	RateLimitNotReachedReason = "RateLimitNotReached"
)
