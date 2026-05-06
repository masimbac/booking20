package dynamo

import "time"

const (
	entityBusiness = "BUSINESS"
	entityService  = "SERVICE"
	entityStaff    = "STAFF"
	entityCustomer = "CUSTOMER"
	entityAvail    = "AVAILABILITY"
	entityBooking  = "BOOKING"
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

func bookingSK(bookingID string) string {
	return "BOOKING#" + bookingID
}

// bookingGSI1SK is the range key on GSI1 (unique per booking via trailing id).
func bookingGSI1SK(startUTC time.Time, bookingID string) string {
	return "BOOKING_DATE#" + startUTC.UTC().Format(time.RFC3339Nano) + "#" + bookingID
}
