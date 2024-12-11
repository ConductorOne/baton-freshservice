package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
)

// By default, the number of objects returned per page is 30.
// The maximum number of objects that can be retrieved per page is 100
// https://developers.freshdesk.com/api/#pagination
const ITEMSPERPAGE = 100

var (
	agentUserResourceType = &v2.ResourceType{
		Id:          "agent",
		DisplayName: "Agent",
		Description: "Agent users of FreshService",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_USER},
	}
	requesterResourceType = &v2.ResourceType{
		Id:          "requester",
		DisplayName: "Requester",
		Description: "Requester users of FreshService",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_USER},
		Annotations: annotations.New(&v2.SkipEntitlementsAndGrants{}),
	}
	agentGroupResourceType = &v2.ResourceType{
		Id:          "agent_group",
		DisplayName: "Agent Group",
		Description: "Agent groups of FreshService",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_GROUP,
		},
	}
	resourceTypeRole = &v2.ResourceType{
		Id:          "role",
		DisplayName: "Role",
		Description: "Roles of FreshService",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_ROLE},
	}
	resourceTypeRequesterGroup = &v2.ResourceType{
		Id:          "requester_group",
		DisplayName: "Requester Group",
		Description: "Requester groups of FreshService",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_GROUP},
	}
)
