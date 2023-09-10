# edx12

Golang X12 EDI parser, validator and JSON converter for 
X12 270, 271 and 835... though it will work for most X12 messages in general,
including envelope validation, but without transaction set-specific validation
outside of those three.

(*warning*: I'm fairly new to Go, so much of this will likely be 
non-idiomatic and/or inefficient for a while.)

## Installation

```bash
$ go get github.com/arcward/edx12
```

## Example

Example message is from: https://x12.org/examples/005010x279/example-1b-response-generic-request-clinic-patients-subscriber-eligibility

```go
package main

import (
	"encoding/json"
	"fmt"
	"github.com/arcward/edx12"
	"log"
)

func main() {
	messageText := `
		ISA*00*Authorizat*00*Security  *ZZ*Interchange Rec*ZZ*Interchange Sen*141001*1037*>*00501*000031033*0*T*:~
		GS*HB*Sample Rec*Sample Sen*20141001*1037*123456*X*005010X279A1~
		ST*271*1234*005010X279A1~
		BHT*0022*11*10001234*20060501*1319~
		HL*1**20*1~
		NM1*PR*2*ABC COMPANY*****PI*842610001~
		HL*2*1*21*1~
		NM1*1P*2*BONE AND JOINT CLINIC*****SV*2000035~
		HL*3*2*22*0~
		TRN*2*93175-012547*9877281234~
		NM1*IL*1*SMITH*JOHN****MI*123456789~
		N3*15197 BROADWAY AVENUE*APT 215~
		N4*KANSAS CITY*MO*64108~
		DMG*D8*19630519*M~
		DTP*346*D8*20060101~
		EB*1**30**GOLD 123 PLAN~
		EB*L~
		EB*1**1>33>35>47>86>88>98>AL>MH>UC~
		EB*B**1>33>35>47>86>88>98>AL>MH>UC*HM*GOLD 123 PLAN*27*10*****Y~
		EB*B**1>33>35>47>86>88>98>AL>MH>UC*HM*GOLD 123 PLAN*27*30*****N~
		LS*2120~
		NM1*P3*1*JONES*MARCUS****SV*0202034~
		LE*2120~
		SE*22*1234~
		GE*1*123456~
		IEA*1*000031033~`
	
	// Load the message, print the ISA control number
	message := edx12.NewMessage()
	if err := edx12.UnmarshalText([]byte(messageText), message); err != nil {
		log.Fatalf("err: %v", err)
	}
	// Prints 'ISA13: 000031033'
	fmt.Printf("ISA13: %s\n", message.ControlNumber())
	
	// Transform the transaction sets in the message and validate them, then
	// print them out as JSON
	for _, txn := range message.TransactionSets() {
		err := txn.Transform()
		if err != nil {
			log.Fatalf("err: %v", err)
		}
		data, err := json.MarshalIndent(txn, "", "  ")
		if err != nil {
			log.Fatalf("err: %v", err)
		}
		// Prints '271/005010X279A1:' followed by the JSON
		fmt.Printf(
			"%s/%s:\n%s\n",
			txn.TransactionSetCode,
			txn.VersionCode,
			string(data),
		)
	}
}
```

JSON output:

```json
{
  "beginningOfHierarchicalTransaction": {
    "Date": "20060501",
    "Time": "1319",
    "hierarchicalStructureCode": "0022",
    "referenceIdentification": "10001234",
    "transactionSetPurposeCode": "11"
  },
  "controlNumber": "1234",
  "header": {
    "controlNumber": "1234",
    "transactionSetCode": "271",
    "versionCode": "005010X279A1"
  },
  "informationSourceLevel": {
    "informationReceiverLevel": {
      "informationReceiverLevel": {
        "hierarchicalChildCode": "1",
        "hierarchicalIdNumber": "2",
        "hierarchicalLevelCode": "21",
        "hierarchicalParentIdNumber": "1"
      },
      "informationReceiverName": {
        "informationReceiverAdditionalIdentification": [],
        "informationReceiverAddress": {},
        "informationReceiverCityStateZipCode": {},
        "informationReceiverName": {
          "entityIdentifierCode": "1P",
          "entityTypeQualifier": "2",
          "identificationCode": "2000035",
          "identificationCodeQualifier": "SV",
          "nameFirst": "",
          "nameLastOrOrganizationName": "BONE AND JOINT CLINIC",
          "nameMiddle": "",
          "nameSuffix": ""
        },
        "informationReceiverProviderInfo": {},
        "informationReceiverRequestValidation": []
      },
      "subscriberLevel": {
        "dependentLevel": {},
        "subscriberLevel": {
          "hierarchicalChildCode": "0",
          "hierarchicalIdNumber": "3",
          "hierarchicalLevelCode": "22",
          "hierarchicalParentIdNumber": "2"
        },
        "subscriberName": {
          "providerInfo": {},
          "subscriberAdditionalIdentification": [],
          "subscriberAddress": {
            "addressInfo00": "15197 BROADWAY AVENUE",
            "addressInfo01": "APT 215"
          },
          "subscriberCityStateZipCode": {
            "cityName": "KANSAS CITY",
            "countryCode": "",
            "countrySubdivisionCode": "",
            "postalCode": "64108",
            "stateOrProvinceCode": "MO"
          },
          "subscriberDate": [
            {
              "dateTimePeriod": "20060101",
              "dateTimePeriodFormatQualifier": "D8",
              "dateTimeQualifier": "346"
            }
          ],
          "subscriberDemographicInfo": {
            "dateTimePeriod": "19630519",
            "dateTimePeriodFormatQualifier": "D8",
            "genderCode": "M"
          },
          "subscriberEligibilityOrBenefitInfo": {
            "healthCareServicesDelivery": [],
            "loopHeader": {
              "loopIdentifierCode": "2120"
            },
            "messageText": [],
            "subscriberAdditionalIdentification": [],
            "subscriberBenefitRelatedEntityName": [
              {
                "loopTrailer": {
                  "loopIdentifierCode": "2120"
                },
                "subscriberBenefitRelatedEntityAddress": {},
                "subscriberBenefitRelatedEntityCityStateZipCode": {},
                "subscriberBenefitRelatedEntityContactInfo": [],
                "subscriberBenefitRelatedEntityName": {
                  "entityIdentifierCode": "P3",
                  "entityRelationshipCode": "",
                  "entityTypeQualifier": "1",
                  "identificationCode": "0202034",
                  "identificationCodeQualifier": "SV",
                  "nameFirst": "MARCUS",
                  "nameLastOrOrganizationName": "JONES",
                  "nameMiddle": "",
                  "nameSuffix": ""
                },
                "subscriberBenefitRelatedProviderInfo": {}
              }
            ],
            "subscriberEligibilityBenefitDate": [],
            "subscriberEligibilityOrBenefitAdditionalInfo": [],
            "subscriberEligibilityOrBenefitInfo": {
              "Quantity": "",
              "compositeDiagnosisCodePointer": {},
              "compositeMedicalProcedureIdentifier": {},
              "coverageLevelCode": "",
              "eligibilityOrBenefitInfoCode": "B",
              "insuranceTypeCode": "HM",
              "monetaryAmount": "30",
              "percentageAsDecimal": "",
              "planCoverageDescription": "GOLD 123 PLAN",
              "quantityQualifier": "",
              "serviceTypeCode": [
                "1",
                "33",
                "35",
                "47",
                "86",
                "88",
                "98",
                "AL",
                "MH",
                "UC"
              ],
              "timePeriodQualifier": "27",
              "yesNoConditionOrResponseCode00": "",
              "yesNoConditionOrResponseCode01": "N"
            },
            "subscriberRequestValidation": []
          },
          "subscriberHealthCareDiagnosisCode": {},
          "subscriberMilitaryPersonnelInfo": {},
          "subscriberName": {
            "entityIdentifierCode": "IL",
            "entityTypeQualifier": "1",
            "identificationCode": "123456789",
            "identificationCodeQualifier": "MI",
            "nameFirst": "JOHN",
            "nameLastOrOrganizationName": "SMITH",
            "nameMiddle": "",
            "nameSuffix": ""
          },
          "subscriberRelationship": {},
          "subscriberRequestValidation": []
        },
        "subscriberTraceNumber": [
          {
            "originatingCompanyIdentifier": "9877281234",
            "referenceIdentification00": "93175-012547",
            "referenceIdentification01": "",
            "traceTypeCode": "2"
          }
        ]
      }
    },
    "informationSourceLevel": {
      "hierarchicalChildCode": "1",
      "hierarchicalIdNumber": "1",
      "hierarchicalLevelCode": "20"
    },
    "informationSourceName": {
      "informationSourceContactInfo": [],
      "informationSourceName": {
        "entityIdentifierCode": "PR",
        "entityTypeQualifier": "2",
        "identificationCode": "842610001",
        "identificationCodeQualifier": "PI",
        "nameFirst": "",
        "nameLastOrOrganizationName": "ABC COMPANY",
        "nameMiddle": "",
        "nameSuffix": ""
      },
      "requestValidation": []
    },
    "requestValidation": []
  },
  "trailer": {
    "controlNumber": "1234",
    "segmentCount": "22"
  },
  "transactionSetCode": "271",
  "version": "005010X279A1"
}

```


