// Package application contains application services, untangled from communication details.
package application

import (
	"authz/domain"
	"authz/domain/contracts"
	"authz/domain/services"
	"context"
)

// AccessAppService the handler for permission related endpoints.
type AccessAppService struct {
	accessRepo    *contracts.AccessRepository
	principalRepo contracts.PrincipalRepository
	ctx           context.Context
}

// CheckRequest is an actual request to check for permissions.
type CheckRequest struct {
	Requestor    string
	Subject      string
	ResourceType string
	ResourceID   string
	Operation    string
}

// NewAccessAppService returns a new instance of the permissionhandler.
func NewAccessAppService(accessRepo *contracts.AccessRepository, principalRepo contracts.PrincipalRepository) *AccessAppService {
	return &AccessAppService{
		accessRepo:    accessRepo,
		principalRepo: principalRepo,
		ctx:           context.Background(),
	}
}

// Check calls the domainservice using a CheckEvent and can be used with every server impl if wanted.
func (p *AccessAppService) Check(req CheckRequest) (domain.AccessDecision, error) {
	event := domain.CheckEvent{
		SubjectID: domain.SubjectID(req.Subject),
		Operation: req.Operation,
		Resource:  domain.Resource{Type: req.ResourceType, ID: req.ResourceID},
	}

	event.Requestor = domain.SubjectID(req.Requestor)

	checkResult := services.NewAccessService(*p.accessRepo)

	return checkResult.Check(event)
}
