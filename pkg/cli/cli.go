package cli

import (
	"bufio"
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"

	"github.com/Fepozopo/timp/pkg/stdimg"
)

func usage() {
	fmt.Println("Commands available:")
	fmt.Println("  /  - select and apply command")
	fmt.Println("  o  - open another image at runtime")
	fmt.Println("  s  - save current image")
	fmt.Println("  u  - check for updates")
	fmt.Println("  h  - show this help message")
	fmt.Println("  q  - quit")
}

func RunCLI() {
	var inputImagePath string
	if len(os.Args) >= 2 {
		inputImagePath = os.Args[1]
	} else {
		inputImagePath = ""
	}

	// Use stdimg command metadata as the canonical source
	storeStd := NewMetaStoreFromStdimg(stdimg.Commands)

	var cur image.Image
	// Track the path of the currently loaded image so we can show EXIF for identify
	var currentImagePath string
	var currentFormat string
	var currentAppSegments []AppSegment
	var currentAutoOriented bool
	if inputImagePath != "" {
		img, format, meta, autoOriented, err := LoadImage(inputImagePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read image %s: %v\n", inputImagePath, err)
			os.Exit(1)
		}
		cur = img
		currentImagePath = inputImagePath
		currentFormat = format
		currentAppSegments = meta
		currentAutoOriented = autoOriented
		// Try to show an initial preview in compatible terminals.
		// Ignore errors here so preview remains optional.
		_ = PreviewImage(cur, currentFormat)
		if info, ierr := GetImageInfoImage(cur); ierr == nil {
			fmt.Println(info)
		}
	}

	fmt.Println("Terminal Image Editor")
	usage()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		r, _, err := reader.ReadRune()
		if err != nil {
			fmt.Fprintf(os.Stderr, "read input error: %v\n", err)
			continue
		}

		switch r {
		case '/':
			if cur == nil {
				fmt.Println("No image loaded. Press 'o' to open an image first, or provide an image path as the first argument.")
				continue
			}
			var commandName string
			name, err := SelectCommandWithFzfStd(stdimg.Commands)
			if err != nil || name == "" {
				// fzf unavailable or returned nothing — fall back to a textual selection list using stdimg.Commands.
				fmt.Println("Command selection (fallback):")
				for i, c := range stdimg.Commands {
					fmt.Printf("  %d) %s - %s\n", i+1, c.Name, c.Description)
				}
				selection, _ := PromptLine("Enter number or command name (leave empty to cancel): ")
				if selection == "" {
					fmt.Println("selection cancelled")
					continue
				}
				if idx, perr := strconv.Atoi(selection); perr == nil {
					if idx < 1 || idx > len(stdimg.Commands) {
						fmt.Println("invalid selection")
						continue
					}
					commandName = stdimg.Commands[idx-1].Name
				} else {
					selLower := strings.ToLower(selection)
					found := ""
					for _, c := range stdimg.Commands {
						if strings.ToLower(c.Name) == selLower {
							found = c.Name
							break
						}
					}
					if found == "" {
						matches := []string{}
						for _, c := range stdimg.Commands {
							if strings.HasPrefix(strings.ToLower(c.Name), selLower) {
								matches = append(matches, c.Name)
							}
						}
						if len(matches) == 1 {
							found = matches[0]
						} else if len(matches) > 1 {
							fmt.Println("ambiguous selection, candidates:")
							for _, m := range matches {
								fmt.Println("  " + m)
							}
							continue
						}
					}
					if found == "" {
						fmt.Printf("unknown command: %s\n", selection)
						continue
					}
					commandName = found
				}
			} else {
				commandName = name
			}

			// Lookup the stdimg command spec
			c, ok := storeStd.byName[commandName]
			if !ok {
				fmt.Printf("unknown command: %s\n", commandName)
				continue
			}

			var rawArgs []string
			// Show tooltip from stdimg metadata
			tooltip, _, _ := storeStd.GetCommandHelp(commandName)
			fmt.Println("\n" + tooltip + "\n")
			rawArgs = make([]string, len(c.Args))
			for i, p := range c.Args {
				// Build prompt label
				typeLabel := p.Type
				if p.Type == "enum" && p.Description != "" {
					typeLabel = fmt.Sprintf("enum(%s)", p.Description)
				}
				prompt := fmt.Sprintf("%s (%s): ", p.Name, typeLabel)

				lowerName := strings.ToLower(p.Name)
				lowerHint := strings.ToLower(p.Description)
				var val string
				var perr error
				if strings.Contains(lowerName, "path") || strings.Contains(lowerName, "file") || strings.Contains(lowerHint, "path") || strings.Contains(lowerHint, "file") {
					prompt = fmt.Sprintf("%s (%s) [enter image path, url, or enter '/' to use fzf]: ", p.Name, typeLabel)
					val, perr = PromptLineWithFzf(prompt)
					if perr != nil {
						fmt.Fprintf(os.Stderr, "input error: %v\n", perr)
						val = ""
					}
				} else {
					val, perr = PromptLine(prompt)
					if perr != nil {
						fmt.Fprintf(os.Stderr, "input error: %v\n", perr)
						val = ""
					}
				}
				rawArgs[i] = val
			}

			normArgs, err := NormalizeArgsFromStd(storeStd, commandName, rawArgs)
			if err != nil {
				fmt.Fprintf(os.Stderr, "input validation error: %v\n", err)
				fmt.Println("aborting command due to input errors")
				continue
			}

			// Apply command using pure-Go stdlib engine
			newImg, err := stdimg.ApplyCommandStdlib(cur, commandName, normArgs)
			if err != nil {
				fmt.Fprintf(os.Stderr, "apply command error: %v\n", err)
				continue
			}
			if newImg != nil {
				cur = newImg
			}
			fmt.Printf("Applied %s\n", commandName)
			_ = PreviewImage(cur, currentFormat)
			if commandName == "strip" {
				// clear stored metadata on strip
				currentAppSegments = nil
				currentAutoOriented = false
				fmt.Println("metadata cleared")
			}
			if commandName == "identify" {
				if currentImagePath != "" {
					if ex, err := ExtractEXIFStruct(currentImagePath); err == nil {
						// Print a concise EXIF summary
						if ex.Make != "" || ex.Model != "" {
							fmt.Printf("Make: %s\nModel: %s\n", ex.Make, ex.Model)
						}
						if ex.Software != "" {
							fmt.Printf("Software: %s\n", ex.Software)
						}
						if ex.Orientation != 0 {
							fmt.Printf("Orientation: %d\n", ex.Orientation)
						}
						if ex.DateTimeOriginal != "" {
							fmt.Printf("DateTimeOriginal: %s\n", ex.DateTimeOriginal)
						}
						if ex.ExposureTime != "" {
							fmt.Printf("ExposureTime: %s sec\n", ex.ExposureTime)
						}
						if ex.Exposure != 0 {
							fmt.Printf("Exposure: %.4f sec\n", ex.Exposure)
						}
						if ex.ShutterSpeed != "" {
							fmt.Printf("ShutterSpeed: %s\n", ex.ShutterSpeed)
						}
						if ex.ApertureValue != 0 {
							fmt.Printf("ApertureValue: f/%.1f\n", ex.ApertureValue)
						}
						if ex.FNumber != 0 {
							fmt.Printf("FNumber: f/%.1f\n", ex.FNumber)
						}
						if ex.MeteringMode != 0 {
							fmt.Printf("MeteringMode: %d\n", ex.MeteringMode)
						}
						if ex.Flash != 0 {
							fmt.Printf("Flash: %d\n", ex.Flash)
						}
						if ex.ISOSpeed != 0 {
							fmt.Printf("ISO Speed: %d\n", ex.ISOSpeed)
						}
						if ex.FocalLength != 0 {
							fmt.Printf("FocalLength: %.1f mm\n", ex.FocalLength)
						}
						if ex.LensModel != "" {
							fmt.Printf("LenseModel: %s\n", ex.LensModel)
						}
						if ex.GPS != nil {
							// Print a nicely formatted GPS summary
							lat := ex.GPS.Latitude
							lon := ex.GPS.Longitude
							latRef := ex.GPS.LatRef
							lonRef := ex.GPS.LonRef
							alt := ex.GPS.Altitude
							altRef := ex.GPS.AltitudeRef

							fmt.Println("GPS:")
							fmt.Printf("  Latitude:  %.8f %s\n", lat, latRef)
							fmt.Printf("  Longitude: %.8f %s\n", lon, lonRef)

							if alt != 0 {
								refStr := "above sea level"
								if altRef == 1 {
									refStr = "below sea level"
								}
								fmt.Printf("  Altitude:  %.2f m (%s)\n", alt, refStr)
							}

							if ex.GPS.GPSDateStamp != "" || ex.GPS.GPSTimeStamp != "" {
								if ex.GPS.GPSDateStamp != "" {
									fmt.Printf("  GPS Date:  %s\n", ex.GPS.GPSDateStamp)
								}
								if ex.GPS.GPSTimeStamp != "" {
									fmt.Printf("  GPS Time:  %s\n", ex.GPS.GPSTimeStamp)
								}
							}

							if !ex.GPS.Timestamp.IsZero() {
								fmt.Printf("  Timestamp: %s\n", ex.GPS.Timestamp.String())
							}
						}
					} else {
						fmt.Fprintf(os.Stderr, "failed to extract EXIF: %v\n", err)
					}
				} else {
					fmt.Println("identify: no image path available to extract EXIF")
				}
			}
			if info, ierr := GetImageInfoImage(cur); ierr == nil {
				fmt.Println(info)
			}
			continue

		case 's':
			out, _ := PromptLine("Enter output filename: ")
			if out == "" {
				fmt.Println("no filename provided")
				continue
			}
			if err := SaveImage(out, cur, currentAppSegments, currentAutoOriented); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write image: %v\n", err)
				continue
			}
			fmt.Printf("Saved to %s\n", out)

		case 'o':
			selected, selErr := SelectFileWithFzf(".")
			var newPath string
			if selErr != nil || selected == "" {
				newPath, _ = PromptLine("Enter path to image to open (leave empty to cancel): ")
				if newPath == "" {
					fmt.Println("open cancelled")
					continue
				}
			} else {
				newPath = selected
			}

			img, format, meta, autoOriented, err := LoadImage(newPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to read image %s: %v\n", newPath, err)
				continue
			}
			cur = img
			currentImagePath = newPath
			currentFormat = format
			currentAppSegments = meta
			currentAutoOriented = autoOriented
			fmt.Printf("Opened %s\n", newPath)
			_ = PreviewImage(cur, currentFormat)
			if info, ierr := GetImageInfoImage(cur); ierr == nil {
				fmt.Println(info)
			}
			continue

		case 'u':
			err := CheckForUpdates()
			if err != nil {
				fmt.Fprintf(os.Stderr, "update check error: %v\n", err)
			}
			continue

		case 'h':
			usage()
			continue

		case 'q':
			fmt.Println("Exiting...")
			return

		default:
			// ignore other keys
		}
	}
}
