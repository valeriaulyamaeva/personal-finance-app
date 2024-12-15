package models

type FamilyAccount struct {
	ID       int    `json:"id"`
	Nickname string `json:"nickname"`
	OwnerID  int    `json:"owner_user_id"`
}

type FamilyMember struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}
