package main

import (
	"image"
	"image/color"
	"strconv"
)

var (
	w = color.White
	y = color.RGBA{200, 200, 0, 0xff}
	z = color.RGBA{0, 0, 0, 0}
	x = color.Black
	p = color.RGBA{138, 0, 196, 0xff}
	r = color.RGBA{0xff, 0, 0, 0xff}
	g = color.RGBA{0x00, 0xb7, 0x19, 0xff}
	l = color.RGBA{0xb3, 0xeb, 0xf2, 0xff}
)

var upArrow = [][]color.Color{
	{r},
	{r, r},
	{r, r, r},
	{r},
	{r},
	{r},
	{r},
	{r},
}

var downArrow = [][]color.Color{
	{r},
	{r},
	{r},
	{r},
	{r},
	{r, r, r},
	{r, r},
	{r},
}

var rBullet = [][]color.Color{
	{z, y, y, y, y, y, y, z},
	{y, y, x, x, x, x, y, y},
	{y, y, x, y, y, x, y, y},
	{y, y, x, x, x, y, y, y},
	{y, y, x, y, x, y, y, y},
	{y, y, x, y, y, x, y, y},
	{y, y, x, y, y, x, y, y},
	{z, y, y, y, y, y, y, z},
}

var fourFiveBullet = [][]color.Color{
	{z, w, g, w, g, g, g, z},
	{g, w, g, w, g, g, g, g},
	{g, w, w, w, g, g, g, g},
	{g, g, g, w, l, l, l, g},
	{g, g, g, w, l, g, g, g},
	{g, g, g, g, l, l, l, g},
	{g, g, g, g, g, g, l, g},
	{z, g, g, g, l, l, l, z},
}

// 5×7 font — per screenshot style
var digits = map[rune][][]color.Color{
	'0': {
		{w, w, w},
		{w, z, w},
		{w, z, w},
		{w, z, w},
		{w, z, w},
		{w, z, w},
		{w, w, w},
	},
	'1': {
		{z, w, z},
		{w, w, z},
		{z, w, z},
		{z, w, z},
		{z, w, z},
		{z, w, z},
		{w, w, w},
	},
	'2': {
		{w, w, w},
		{z, z, w},
		{z, z, w},
		{w, w, w},
		{w, z, z},
		{w, z, z},
		{w, w, w},
	},
	'3': {
		{w, w, w},
		{z, z, w},
		{z, z, w},
		{w, w, w},
		{z, z, w},
		{z, z, w},
		{w, w, w},
	},
	'4': {
		{w, z, w},
		{w, z, w},
		{w, z, w},
		{w, w, w},
		{z, z, w},
		{z, z, w},
		{z, z, w},
	},
	'5': {
		{w, w, w},
		{w, z, z},
		{w, z, z},
		{w, w, w},
		{z, z, w},
		{z, z, w},
		{w, w, w},
	},
	'6': {
		{w, w, w},
		{w, z, z},
		{w, z, z},
		{w, w, w},
		{w, z, w},
		{w, z, w},
		{w, w, w},
	},
	'7': {
		{w, w, w},
		{z, z, w},
		{z, z, w},
		{z, z, w},
		{z, w, z},
		{z, w, z},
		{z, w, z},
	},
	'8': {
		{w, w, w},
		{w, z, w},
		{w, z, w},
		{w, w, w},
		{w, z, w},
		{w, z, w},
		{w, w, w},
	},
	'9': {
		{w, w, w},
		{w, z, w},
		{w, z, w},
		{w, w, w},
		{z, z, w},
		{z, z, w},
		{w, w, w},
	},
	' ': {
		{z, z, z},
		{z, z, z},
		{z, z, z},
		{z, z, z},
		{z, z, z},
		{z, z, z},
		{z, z, z},
	},
	',': {
		{z, z, z},
		{z, z, z},
		{z, z, z},
		{z, z, z},
		{z, z, z},
		{z, w, z},
		{w, z, z},
	},
}

func drawSprite(img *image.RGBA, sprite [][]color.Color, ox, oy int) {
	for y := 0; y < len(sprite); y++ {
		for x := 0; x < len(sprite[y]); x++ {
			c := sprite[y][x]
			if c == nil {
				continue
			}
			img.Set(ox+x, oy+y, c)
		}
	}
}

func format2(n int) string {
	if n < 10 {
		return strconv.Itoa(n)
	}
	if n > 99 {
		return "99"
	}
	return strconv.Itoa(n)
}
