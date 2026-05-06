package dynamo

const (
	entityBusiness = "BUSINESS"
	entityService  = "SERVICE"
	entityStaff    = "STAFF"
	entityCustomer = "CUSTOMER"
	entityAvail    = "AVAILABILITY"
)

// BusinessPK returns the partition key for tenant-scoped data.
func BusinessPK(businessID string) string {
	return "BUSINESS#" + businessID
}

// BusinessMetaSK is the sort key for the business aggregate row.
func BusinessMetaSK(businessID string) string {
	return "META#" + businessID
}

func serviceSK(serviceID string) string {
	return "SERVICE#" + serviceID
}

func staffSK(staffID string) string {
	return "STAFF#" + staffID
}

func customerSK(customerID string) string {
	return "CUSTOMER#" + customerID
}

func availRulesSK() string {
	return "AVAIL#RULES"
}

func phoneGSI2PK(e164 string) string {
	return "PHONE#" + e164
}

func phoneGSI2SK(businessID string) string {
	return "BUSINESS#" + businessID
}
