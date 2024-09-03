package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

func main() {

    img := flag.String("img", "", "Image to extract info from")
    wantGoogleMap := flag.Bool("gmap", false, "Indicate if you want to create a Google Map link with the GPS infos there is some")
    toJson := flag.Bool("json", false, "If you want the output to be in JSON")

    flag.Parse()

    if *img == "" {
        flag.PrintDefaults()
        return
    }

	file, err := os.Open(*img)
	if err != nil {
		fmt.Println("[*] Failed to open the image in arguments")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("[*] Failed to read the content of the image")
		return
	}

	exifData, err := exif.SearchAndExtractExif(content)
	if err != nil {
		fmt.Println("[*] Failed to extract/find exif in the image")
		return
	}

	index, _, err := exif.GetFlatExifData(exifData, nil)

	var gpsLatitude, gpsLongitude float64
    hasGPSData := false

    if *toJson {
        handleJson(index, wantGoogleMap)
    } else {
   
        for _, entry := range index {

            // Print all EXIF data
            val, err := formatValue(entry.Value)
            if err != nil {
                fmt.Printf("%s: %v\n", entry.TagName, entry.Value)
                fmt.Printf("Failed to format, type: %s\n",reflect.TypeOf(entry.Value))
            } else {
                fmt.Printf("%s: %s\n", entry.TagName, val)
            }

            if *wantGoogleMap {
                // Extract GPSLatitude and GPSLongitude
                if entry.TagName == "GPSLatitude" {
                    gpsLatitude = parseGPS(entry.Value.([]exifcommon.Rational))
                    hasGPSData = true
                } else if entry.TagName == "GPSLongitude" {
                    gpsLongitude = parseGPS(entry.Value.([]exifcommon.Rational))
                    hasGPSData = true
                } else if entry.TagName == "GPSLatitudeRef" && entry.Value.(string) == "S" {
                    gpsLatitude = -gpsLatitude
                } else if entry.TagName == "GPSLongitudeRef" && entry.Value.(string) == "W" {
                    gpsLongitude = -gpsLongitude
                }

            }
        }

        if hasGPSData {
            // Generate and print the Google Maps link
            googleMapsLink := fmt.Sprintf("https://www.google.com/maps?q=%f,%f", gpsLatitude, gpsLongitude)
            fmt.Printf("Google Maps Link: %s\n", googleMapsLink)
        }   

    }

}


func handleJson(exifData []exif.ExifTag, wantGoogleMap *bool) {
    exifMap := make(map[string]string)
    hasGPSData := false
    var gpsLatitude, gpsLongitude float64
    
    for _, entry := range exifData {
        val, err := formatValue(entry.Value)
        if err != nil {
            exifMap[entry.TagName] = fmt.Sprintf("%v", entry.Value)
        } else {
            exifMap[entry.TagName] = val
        }

        if *wantGoogleMap {
            
            // Extract GPSLatitude and GPSLongitude
            if entry.TagName == "GPSLatitude" {
                gpsLatitude = parseGPS(entry.Value.([]exifcommon.Rational))
                hasGPSData = true
            } else if entry.TagName == "GPSLongitude" {
                gpsLongitude = parseGPS(entry.Value.([]exifcommon.Rational))
                hasGPSData = true
            } else if entry.TagName == "GPSLatitudeRef" && entry.Value.(string) == "S" {
                gpsLatitude = -gpsLatitude
            } else if entry.TagName == "GPSLongitudeRef" && entry.Value.(string) == "W" {
                gpsLongitude = -gpsLongitude
            }

        }


    }

    if hasGPSData {
        exifMap["GoogleMapsLink"] = fmt.Sprintf("https://www.google.com/maps?q=%f,%f", gpsLatitude, gpsLongitude)
    }

    jsonString, err := json.MarshalIndent(exifMap, "", "  ")
    if err != nil {
        fmt.Println("[*] Failed to marshall the exifMap into JSON")
        return
    }

    fmt.Printf("%s\n", jsonString)
}

// parseGPS converts GPS coordinates from degrees/minutes/seconds (DMS) to decimal
func parseGPS(rationals []exifcommon.Rational) float64 {
	degrees := float64(rationals[0].Numerator) / float64(rationals[0].Denominator)
	minutes := float64(rationals[1].Numerator) / float64(rationals[1].Denominator) / 60.0
	seconds := float64(rationals[2].Numerator) / float64(rationals[2].Denominator) / 3600.0
	return degrees + minutes + seconds
}

func formatValue(value interface{}) (string, error) {
    switch v := value.(type) {
    case []exifcommon.Rational:
        if len(v) == 3 {
            return strconv.FormatFloat(parseGPS(v), 'f', -1, 64), nil
	    } else if len(v) == 1 {
            return strconv.FormatFloat(float64(v[0].Numerator) / float64(v[0].Denominator), 'f', -1, 64), nil
        }

    case []exifcommon.SignedRational:
        if len(v) == 1 {
            return strconv.FormatFloat(float64(v[0].Numerator) / float64(v[0].Denominator), 'f', -1, 64), nil
        }
    
	case []uint8:
		return formatUintSlice(v) 
	case []uint16:
		return formatUintSlice(v)
	case []uint32:
		return formatUintSlice(v)
	case []uint64:
		return formatUintSlice(v)

    case []int:
        if len(v) == 1 {
            return strconv.Itoa(v[0]), nil
        }

    default:
        return fmt.Sprintf("%v", v), nil
    }
    return "", errors.New("Failed to convert")
}

func formatUintSlice[T uint8 | uint16 | uint32 | uint64](slice []T) (string, error) {
	if len(slice) == 1 {
		return strconv.FormatUint(uint64(slice[0]), 10), nil
	} else if len(slice) == 2 {
        return strconv.FormatFloat(float64(slice[0]) / float64(slice[1]), 'f', -1, 64), nil
    }
    return fmt.Sprintf("%v", slice), nil
}
