// Package iso builds the SWL ISO directory structure using xorriso.
package iso

import (
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/host90-gist/RNS510-Bootlogo-GO/internal/fli"
)

func Build(img image.Image, outputISO, templateDir, fit string) error {
	templateDir, err := filepath.Abs(templateDir)
	if err != nil { return err }
	required := []string{
		filepath.Join(templateDir, "BLSCRIPT.CFG"),
		filepath.Join(templateDir, "VERSION.TXT"),
		filepath.Join(templateDir, "SWL", "SILVER_1", "RNSMIDEC", "SWL", "SWL.FLI"),
	}
	for _, p := range required {
		if _, err := os.Stat(p); err != nil { return fmt.Errorf("required SWL template file %q: %w", p, err) }
	}
	fliData, err := fli.Encode(img, fit, true)
	if err != nil { return err }
	if _, err := fli.Verify(fliData); err != nil { return fmt.Errorf("generated LOGO.FLI failed verification: %w", err) }

	stage, err := os.MkdirTemp("", "rns510-swl-*")
	if err != nil { return err }
	defer os.RemoveAll(stage)
	copyFile := func(src, dst string) error {
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil { return err }
		in, err := os.Open(src); if err != nil { return err }; defer in.Close()
		out, err := os.Create(dst); if err != nil { return err }
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil { return copyErr }; return closeErr
	}
	for _, item := range []struct{src, dst string}{
		{filepath.Join(templateDir,"BLSCRIPT.CFG"), filepath.Join(stage,"BLSCRIPT.CFG")},
		{filepath.Join(templateDir,"VERSION.TXT"), filepath.Join(stage,"VERSION.TXT")},
		{filepath.Join(templateDir,"DIR.INF"), filepath.Join(stage,"DIR.INF")},
		{filepath.Join(templateDir,"SWL","SILVER_1","RNSMIDEC","SWL","SWL.FLI"), filepath.Join(stage,"SWL","SILVER_1","RNSMIDEC","SWL","SWL.FLI")},
	} {
		if _, err := os.Stat(item.src); err == nil {
			if err := copyFile(item.src,item.dst); err != nil { return err }
		} else if filepath.Base(item.src) == "DIR.INF" {
			if err := os.WriteFile(item.dst, []byte("\r\n"), 0644); err != nil { return err }
		}
	}
	if err := os.MkdirAll(filepath.Join(stage,"WA"),0755); err != nil { return err }
	if err := os.WriteFile(filepath.Join(stage,"WA","LOGO.FLI"),fliData,0644); err != nil { return err }
	ecu := []byte(fmt.Sprintf("SHOW_SCREEN 0 1 SWL_SCREEN_VERSION\r\nSTRATA_UPDATE 15 %d /cddos/WA/LOGO.FLI\r\nRESTART_SYSTEM\r\nFINISHED_DOWNLOAD\r\n",len(fliData)))
	if err := os.WriteFile(filepath.Join(stage,"ECUORDER.TXT"),ecu,0644); err != nil { return err }
	crcText := []byte(fmt.Sprintf("%04x",fli.CRC16XMODEM(ecu)))
	if err := os.WriteFile(filepath.Join(stage,"CRC.16"),crcText,0644); err != nil { return err }
	outputISO, err = filepath.Abs(outputISO)
	if err != nil { return err }
	if err := os.MkdirAll(filepath.Dir(outputISO),0755); err != nil { return err }
	if _, err := exec.LookPath("xorriso"); err != nil { return fmt.Errorf("xorriso is required to build ISO images; install it or use the packaged AppImage") }
	cmd := exec.Command("xorriso","-as","mkisofs","-iso-level","1","-sysid","Win32","-V","SWL_LOGO","-o",outputISO,stage)
	if output, err := cmd.CombinedOutput(); err != nil { return fmt.Errorf("xorriso failed: %w: %s",err,string(output)) }
	return nil
}
