package access

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-/gocryptotrader/gctrpc"
)

// Consts here are roles for usage within go crypto trader
const (
	RoleFinancialAnalyst Permission = 1 << iota
	RoleTrader
	RoleRegulatoryComplianceOfficer
	RoleQuantitativeAnalyst
	RoleMarketing
	RolePortfolioManager
	RoleJuniorSoftwareDeveloper
	RoleSoftwareDeveloper
	RoleSeniorDeveloper
	RoleChiefTechnicalOfficer
	RoleAccountant
	RoleOperations
	RoleSales
	RoleClient
	RoleManager
	RoleTeamLead
	RoleRiskOfficer

	// Critical level role TODO: alert system admins to access
	RoleSuperUser

	FinancialAnalyst            = "FinancialAnalyst"
	Trader                      = "Trader"
	RegulatoryComplianceOfficer = "RegulatoryComplianceOfficer"
	QuantitativeAnalyst         = "QuantitativeAnalyst"
	Marketing                   = "Marketing"
	PortfolioManager            = "PortfolioManager"
	JuniorSoftwareDeveloper     = "JuniorSoftwareDeveloper"
	SoftwareDeveloper           = "SoftwareDeveloper"
	SeniorSoftwareDeveloper     = "SeniorSoftwareDeveloper"
	ChiefTechnicalOfficer       = "ChiefTechnicalOfficer"
	Accountant                  = "Accountant"
	Operations                  = "Operations"
	Sales                       = "Sales"
	Client                      = "Client"
	Manager                     = "Manager"
	TeamLead                    = "TeamLead"
	RiskOfficer                 = "RiskOfficer"
)

// Consts here define function permission sets
const (
	PermissionAddClient = RoleSales | RoleManager | RoleTeamLead | RoleChiefTechnicalOfficer
)

// Permission defines roles allowable for function use
type Permission uint64

// Role defines a context value access type
var Role Permission

// ClientRoles defines the entire client role list in this package
var ClientRoles = []string{
	FinancialAnalyst,
	Trader,
	RegulatoryComplianceOfficer,
	QuantitativeAnalyst,
	Marketing,
	PortfolioManager,
	JuniorSoftwareDeveloper,
	SoftwareDeveloper,
	SeniorSoftwareDeveloper,
	ChiefTechnicalOfficer,
	Accountant,
	Operations,
	Sales,
	Client,
	Manager,
	TeamLead,
	RiskOfficer,
}

// CheckPermission checks context permission value against
func CheckPermission(ctx context.Context, p Permission) error {
	r, ok := ctx.Value(Role).(Permission)
	if !ok {
		return errors.New("role not set correctly for client")
	}

	if r&RoleSuperUser != 0 {
		return nil
	}

	if r&p != 0 {
		return nil
	}

	return fmt.Errorf("access denied")
}

// GetClientPermission converts client role strings to permission bitmask
func GetClientPermission(roles []string) Permission {
	var p Permission
	for i := range roles {
		switch roles[i] {
		case FinancialAnalyst:
			p |= RoleFinancialAnalyst
		case Trader:
			p |= RoleTrader
		case RegulatoryComplianceOfficer:
			p |= RoleRegulatoryComplianceOfficer
		case QuantitativeAnalyst:
			p |= RoleQuantitativeAnalyst
		case Marketing:
			p |= RoleMarketing
		case PortfolioManager:
			p |= RolePortfolioManager
		case JuniorSoftwareDeveloper:
			p |= RoleJuniorSoftwareDeveloper
		case SoftwareDeveloper:
			p |= RoleSoftwareDeveloper
		case SeniorSoftwareDeveloper:
			p |= RoleSeniorDeveloper
		case ChiefTechnicalOfficer:
			p |= RoleChiefTechnicalOfficer
		case Accountant:
			p |= RoleAccountant
		case Operations:
			p |= RoleOperations
		case Sales:
			p |= RoleSales
		case Client:
			p |= RoleClient
		case Manager:
			p |= RoleManager
		case TeamLead:
			p |= RoleTeamLead
		case RiskOfficer:
			p |= RoleRiskOfficer
		}
	}
	return p
}

// GetClientDatabaseRoleStrings returns the strings of desired roles for
// entering into database
func GetClientDatabaseRoleStrings(r *gctrpc.Roles) []string {
	var roles []string
	if r == nil {
		return nil
	}

	if r.Accounting {
		roles = append(roles, Accountant)
	}
	if r.FinancialAnalyst {
		roles = append(roles, FinancialAnalyst)
	}
	if r.Trader {
		roles = append(roles, Trader)
	}
	if r.RegulatoryComplianceOfficer {
		roles = append(roles, RegulatoryComplianceOfficer)
	}
	if r.QuantitativeAnalyst {
		roles = append(roles, QuantitativeAnalyst)
	}
	if r.Marketing {
		roles = append(roles, Marketing)
	}
	if r.PortfolioManager {
		roles = append(roles, PortfolioManager)
	}
	if r.JuniorSoftwareDeveloper {
		roles = append(roles, JuniorSoftwareDeveloper)
	}
	if r.SoftwareDeveloper {
		roles = append(roles, SoftwareDeveloper)
	}
	if r.SeniorDeveloper {
		roles = append(roles, SeniorSoftwareDeveloper)
	}
	if r.Operations {
		roles = append(roles, Operations)
	}
	if r.Sales {
		roles = append(roles, Sales)
	}
	if r.Client {
		roles = append(roles, Client)
	}
	if r.Manager {
		roles = append(roles, Manager)
	}
	if r.TeamLead {
		roles = append(roles, TeamLead)
	}
	if r.RiskOfficer {
		roles = append(roles, RiskOfficer)
	}
	if r.ChiefTechnicalOfficer {
		roles = append(roles, ChiefTechnicalOfficer)
	}
	return roles
}
