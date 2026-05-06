package dynamo

const (
	entityBusiness = "BUSINESS"
	entityService  = "SERVICE"
	entityStaff    = "STAFF"
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
