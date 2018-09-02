package bilateral_test

import (
	"image/color"
	_ "image/jpeg"
	"reflect"
	"testing"

	"github.com/mdouchement/bilateral"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func TestFastBilateralColor(t *testing.T) {
	mi := images["base"]
	mo := images["filtered"]

	filter := bilateral.Auto(mi)
	filter.Execute()

	if filter.ColorModel() != color.RGBAModel {
		t.Errorf("%s: expected: %#v, actual: %#v", "ColorModel", color.RGBAModel, filter.ColorModel())
	}

	if !reflect.DeepEqual(filter.Bounds(), mi.Bounds()) {
		t.Errorf("%s: expected: %#v, actual: %#v", "Bounds", mi.Bounds(), filter.Bounds())
	}

	if !reflect.DeepEqual(filter.ResultImage(), mo) {
		t.Errorf("%s: expected: %#v, actual: %#v", "ResultImage", mo, filter.ResultImage())
	}

	for y := 0; y < mi.Bounds().Dy(); y++ {
		for x := 0; x < mi.Bounds().Dx(); x++ {
			if !reflect.DeepEqual(filter.At(x, y), mo.At(x, y)) {
				t.Errorf("%s(%d,%d): expected: %#v, actual: %#v", "At", x, y, mo.At(x, y), filter.At(x, y))
			}
		}
	}
}

func TestFastBilateralGray(t *testing.T) {
	mi := images["base-gray"]
	mo := images["base-gray-filtered"]

	filter := bilateral.Auto(mi)
	filter.Execute()

	if filter.ColorModel() != color.RGBAModel {
		t.Errorf("%s: expected: %#v, actual: %#v", "ColorModel", color.RGBAModel, filter.ColorModel())
	}

	if !reflect.DeepEqual(filter.Bounds(), mi.Bounds()) {
		t.Errorf("%s: expected: %#v, actual: %#v", "Bounds", mi.Bounds(), filter.Bounds())
	}

	if !reflect.DeepEqual(filter.ResultImage(), mo) {
		t.Errorf("%s: expected: %#v, actual: %#v", "ResultImage", mo, filter.ResultImage())
	}

	for y := 0; y < mi.Bounds().Dy(); y++ {
		for x := 0; x < mi.Bounds().Dx(); x++ {
			if !reflect.DeepEqual(filter.At(x, y), mo.At(x, y)) {
				t.Errorf("%s(%d,%d): expected: %#v, actual: %#v", "At", x, y, mo.At(x, y), filter.At(x, y))
			}
		}
	}
}
