package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

// By default, the number of objects returned per page is 30.
// The maximum number of objects that can be retrieved per page is 100
// https://developers.freshdesk.com/api/#pagination
const ITEMSPERPAGE = 100

var (
	userResourceType = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Description: "Users of FreshService",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_USER},
	}

	groupResourceType = &v2.ResourceType{
		Id:          "group",
		DisplayName: "Group",
		Description: "Groups of FreshService",
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
		Description: "Requester Group of FreshService",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_GROUP},
	}
)
