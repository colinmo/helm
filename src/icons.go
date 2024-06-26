package main

import (
	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

var CloudConnect = widget.NewIcon(
	fyne.NewStaticResource("cloudconnect.png", []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x5a, 0x00, 0x00, 0x00, 0x5a,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x38, 0xa8, 0x41, 0x02, 0x00, 0x00, 0x00,
		0x06, 0x62, 0x4b, 0x47, 0x44, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0xa0,
		0xbd, 0xa7, 0x93, 0x00, 0x00, 0x02, 0xf1, 0x49, 0x44, 0x41, 0x54, 0x78,
		0x9c, 0xed, 0xda, 0xcb, 0x6b, 0x1d, 0x55, 0x1c, 0x07, 0xf0, 0xcf, 0x8d,
		0xd6, 0x1a, 0xdd, 0xd4, 0xd6, 0x4d, 0x7d, 0x55, 0x2d, 0x28, 0x54, 0x90,
		0x10, 0x74, 0x51, 0x8b, 0x88, 0x50, 0x45, 0xc5, 0x47, 0xdc, 0x17, 0xdd,
		0x14, 0xb4, 0x0b, 0x53, 0x50, 0x48, 0x17, 0xfe, 0x01, 0xdd, 0x89, 0x76,
		0xe3, 0x42, 0xa8, 0x20, 0xbe, 0x15, 0xa4, 0xa8, 0x2b, 0xa9, 0x1b, 0x05,
		0x1f, 0x28, 0x3e, 0xa0, 0x88, 0x1b, 0x2b, 0x21, 0x2a, 0x55, 0x2b, 0xd8,
		0x87, 0x4d, 0x52, 0x9b, 0x9f, 0x8b, 0x39, 0x39, 0xde, 0x8a, 0xa1, 0x20,
		0xb5, 0x87, 0x3b, 0xfc, 0x3e, 0x97, 0x0b, 0x97, 0x99, 0x73, 0xe1, 0xcb,
		0x97, 0xc3, 0x99, 0x99, 0xc3, 0x90, 0x52, 0x4a, 0x29, 0xa5, 0x94, 0x52,
		0x4a, 0x29, 0xa5, 0x94, 0x52, 0x4a, 0x29, 0xa5, 0x94, 0x52, 0x4a, 0x29,
		0xa5, 0x94, 0x52, 0x4a, 0x29, 0xa5, 0x94, 0x52, 0x4a, 0x29, 0xa5, 0x94,
		0x52, 0x4a, 0x29, 0xa5, 0x94, 0x52, 0x4a, 0x1b, 0xf0, 0x48, 0xeb, 0x10,
		0x7d, 0x37, 0x8e, 0xcf, 0x71, 0x0a, 0x53, 0x8d, 0xb3, 0xf4, 0xda, 0x0b,
		0x88, 0xf2, 0x3d, 0x8e, 0x9b, 0xda, 0xc6, 0xe9, 0xa7, 0x69, 0x84, 0x8b,
		0x85, 0xfb, 0x6a, 0xd9, 0x3f, 0xe2, 0xd2, 0xb6, 0xb1, 0xfa, 0xe5, 0x36,
		0x2c, 0x1a, 0x08, 0xaf, 0x08, 0xf7, 0xd6, 0xa2, 0xf7, 0x63, 0xac, 0x6d,
		0xb4, 0xfe, 0xb8, 0x12, 0x87, 0x10, 0x76, 0x09, 0x7b, 0x6a, 0xc9, 0xbf,
		0xe1, 0xaa, 0xa6, 0xc9, 0x7a, 0xe4, 0x42, 0x7c, 0x8a, 0xb0, 0x55, 0xf8,
		0x52, 0x18, 0xaf, 0x45, 0x3f, 0xd0, 0x36, 0x5a, 0xbf, 0x3c, 0x8f, 0x70,
		0xad, 0x30, 0x27, 0xdc, 0x50, 0x4b, 0x7e, 0xb6, 0x71, 0xae, 0x5e, 0x99,
		0xb1, 0x7c, 0xf1, 0xfb, 0x4a, 0xd8, 0x51, 0x4b, 0x3e, 0xa0, 0xbb, 0xcd,
		0x4b, 0x67, 0xc1, 0x3d, 0xf8, 0xd3, 0x40, 0x78, 0x4d, 0x78, 0x57, 0x18,
		0x08, 0xcc, 0x63, 0xa2, 0x71, 0xb6, 0xde, 0x98, 0xc0, 0x51, 0x84, 0xdd,
		0xc2, 0xac, 0xb0, 0xae, 0xce, 0xe6, 0x9d, 0x6d, 0xa3, 0xf5, 0xc7, 0x7a,
		0xcc, 0x22, 0x3c, 0x2c, 0x2c, 0x0a, 0xb7, 0xd4, 0x92, 0xdf, 0xc1, 0xa0,
		0x69, 0xba, 0x9e, 0x18, 0xc7, 0xc7, 0x08, 0x5b, 0x84, 0x79, 0xe1, 0x89,
		0x5a, 0xf2, 0xac, 0x7c, 0x30, 0x39, 0x2b, 0xc6, 0xf0, 0x26, 0xc2, 0x35,
		0xc2, 0xcf, 0xc2, 0xbe, 0xba, 0x2e, 0x2f, 0x62, 0x73, 0xdb, 0x78, 0xfd,
		0xb1, 0x07, 0x61, 0x8d, 0x70, 0x40, 0x38, 0x28, 0x5c, 0x52, 0x67, 0xf3,
		0xe3, 0x8d, 0xb3, 0xf5, 0xc6, 0x2e, 0x84, 0x0b, 0x84, 0xf7, 0xca, 0xba,
		0xbc, 0xb9, 0x96, 0xfc, 0xb6, 0x5c, 0x97, 0x57, 0x74, 0x27, 0xf6, 0xe9,
		0xd6, 0xdb, 0xbb, 0xcf, 0x30, 0x76, 0x1b, 0x96, 0x8c, 0x09, 0xaf, 0x0b,
		0x21, 0x3c, 0x56, 0x4b, 0xfe, 0x0e, 0x6b, 0xfe, 0xdf, 0xa8, 0xa3, 0x6b,
		0x5a, 0xb7, 0x57, 0xdc, 0x95, 0x35, 0x70, 0x02, 0x57, 0xac, 0x30, 0x76,
		0x2b, 0x16, 0x10, 0x9e, 0x29, 0x25, 0xbf, 0x54, 0x4b, 0x9e, 0x97, 0x5b,
		0xa0, 0x2b, 0x9a, 0xb2, 0x3c, 0x3b, 0x77, 0x0b, 0x0f, 0xd5, 0xd2, 0xf6,
		0xfe, 0xcb, 0xd8, 0x49, 0x1c, 0x41, 0x98, 0x29, 0x25, 0x7f, 0x21, 0x5c,
		0x54, 0xff, 0xf3, 0xe8, 0xb9, 0x8b, 0x3d, 0x7a, 0xf6, 0x1b, 0x9e, 0x9d,
		0xdf, 0x0b, 0xe7, 0xd5, 0xd9, 0xb9, 0x6e, 0x68, 0xdc, 0x26, 0xfc, 0x82,
		0xb0, 0x4d, 0x58, 0x12, 0x7e, 0x15, 0xae, 0xae, 0x25, 0x3f, 0x77, 0xee,
		0xa3, 0x8f, 0x96, 0x6e, 0x97, 0xed, 0xc3, 0x52, 0x74, 0x08, 0x77, 0xd5,
		0xf2, 0xa6, 0xcb, 0x98, 0x8d, 0xf8, 0x81, 0xb2, 0x79, 0xbf, 0x28, 0x9c,
		0x2c, 0x3b, 0x73, 0xdd, 0xb8, 0x8f, 0xb0, 0xba, 0x49, 0xfa, 0x11, 0xf2,
		0x14, 0xc2, 0xce, 0xa1, 0xa2, 0xdf, 0xa8, 0x05, 0x7e, 0xad, 0xdb, 0x57,
		0x3e, 0x88, 0x70, 0x87, 0x70, 0xa2, 0x8c, 0xf9, 0xfb, 0xa1, 0xe4, 0x27,
		0x5c, 0xde, 0x2c, 0xfd, 0x08, 0x99, 0x44, 0x58, 0x2b, 0x1c, 0x2b, 0x25,
		0x2e, 0x08, 0x97, 0xd5, 0x22, 0xbb, 0xcd, 0xfb, 0x5b, 0x85, 0xe3, 0xe5,
		0xfc, 0xcb, 0xf5, 0xdc, 0x02, 0xb6, 0x34, 0xcc, 0x3e, 0x72, 0x3e, 0x30,
		0xbc, 0x4e, 0x87, 0xf0, 0x74, 0x2d, 0x33, 0xdc, 0x2c, 0xfc, 0x5e, 0x8e,
		0x7f, 0x76, 0xda, 0xc5, 0x6f, 0x47, 0xd3, 0xd4, 0x23, 0xe8, 0x7e, 0xca,
		0x85, 0xed, 0x64, 0x29, 0xf4, 0x0f, 0x61, 0xbd, 0x70, 0xa3, 0x70, 0xb8,
		0x1c, 0x9b, 0x3b, 0x6d, 0xa6, 0xe7, 0xc5, 0xef, 0x3f, 0x18, 0xc3, 0x37,
		0x94, 0x65, 0x61, 0x78, 0xad, 0x3e, 0x54, 0x7e, 0x1f, 0x13, 0x26, 0x6b,
		0xc9, 0xef, 0x63, 0x55, 0xd3, 0xc4, 0x23, 0x6c, 0x3b, 0xc2, 0x44, 0xb9,
		0x75, 0x1b, 0xfe, 0x9c, 0x12, 0xa6, 0x6a, 0xc9, 0xdf, 0x62, 0x6d, 0xdb,
		0xa8, 0xa3, 0x6d, 0x35, 0xe6, 0x10, 0x5e, 0xfd, 0x47, 0xd1, 0x33, 0xb5,
		0xe4, 0xc3, 0xb8, 0xae, 0x69, 0xca, 0x9e, 0xe8, 0x66, 0xf5, 0x06, 0xe1,
		0x48, 0x29, 0x79, 0x6f, 0x2d, 0x79, 0x11, 0xb7, 0xb7, 0x8d, 0xd7, 0x1f,
		0xe7, 0xe3, 0x13, 0x84, 0x07, 0x75, 0xef, 0x62, 0xac, 0xaa, 0x45, 0x6f,
		0x6f, 0x1b, 0xad, 0x7f, 0xae, 0xb7, 0xfc, 0xa8, 0xdd, 0x7d, 0x97, 0xf0,
		0x64, 0xd3, 0x44, 0x3d, 0xb6, 0x09, 0x2f, 0xe2, 0x2d, 0xf9, 0xc2, 0x4b,
		0x4a, 0x29, 0xa5, 0x94, 0x52, 0x4a, 0x29, 0xa5, 0x94, 0x52, 0x4a, 0x29,
		0xa5, 0x9e, 0xfa, 0x0b, 0xcb, 0xa1, 0x55, 0x6d, 0xd9, 0xdd, 0x6b, 0xc1,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}),
)
var CloudDisconnect = widget.NewIcon(
	fyne.NewStaticResource("clouddisconnect.png", []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x5a, 0x00, 0x00, 0x00, 0x5a,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x38, 0xa8, 0x41, 0x02, 0x00, 0x00, 0x00,
		0x06, 0x62, 0x4b, 0x47, 0x44, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0xa0,
		0xbd, 0xa7, 0x93, 0x00, 0x00, 0x03, 0xbe, 0x49, 0x44, 0x41, 0x54, 0x78,
		0x9c, 0xed, 0xda, 0x5f, 0x88, 0x54, 0x55, 0x18, 0x00, 0xf0, 0xdf, 0x3a,
		0xee, 0xba, 0xbb, 0x33, 0xeb, 0xee, 0xfa, 0x27, 0xa2, 0xcc, 0x24, 0x21,
		0x22, 0xcb, 0x84, 0x20, 0x7c, 0x28, 0x29, 0x7b, 0x28, 0x88, 0x28, 0xd0,
		0xea, 0xa1, 0x5e, 0xd2, 0x7a, 0xaa, 0x08, 0xa5, 0x12, 0x0a, 0x7a, 0x4b,
		0xa2, 0x8c, 0xa0, 0x30, 0xa8, 0x87, 0xd0, 0x20, 0x7a, 0xc8, 0x8a, 0x4a,
		0x0a, 0x83, 0x40, 0x28, 0x22, 0x7c, 0x28, 0x29, 0x22, 0x2b, 0xc1, 0xca,
		0x22, 0xa5, 0x36, 0xd7, 0x75, 0xfe, 0xac, 0xba, 0xb3, 0xa7, 0x87, 0xab,
		0x64, 0xdb, 0x9d, 0x5d, 0xc9, 0xdc, 0xd9, 0x59, 0xbf, 0x1f, 0xdc, 0x97,
		0x99, 0xef, 0xce, 0x7c, 0xe7, 0x9b, 0xcb, 0x77, 0xce, 0x3d, 0x77, 0x08,
		0x21, 0x84, 0x10, 0x42, 0x08, 0x21, 0x84, 0x10, 0x42, 0x08, 0x21, 0x84,
		0x10, 0x42, 0x08, 0x21, 0x84, 0x10, 0x42, 0x08, 0x21, 0x84, 0x10, 0x42,
		0x08, 0x21, 0x84, 0x10, 0x42, 0x08, 0x21, 0x84, 0x10, 0x42, 0x08, 0x21,
		0x84, 0x10, 0x42, 0x08, 0x21, 0x84, 0x10, 0x42, 0x08, 0x21, 0x84, 0x10,
		0xce, 0x9e, 0x42, 0xb3, 0x13, 0x68, 0x21, 0xf3, 0x67, 0x70, 0x7f, 0xa2,
		0x6d, 0x26, 0xf7, 0x8d, 0xf2, 0x2b, 0xfe, 0x6c, 0x76, 0x52, 0xad, 0x60,
		0x0e, 0x2e, 0xee, 0x66, 0x63, 0x17, 0xeb, 0x27, 0x88, 0x5d, 0xdc, 0xc3,
		0xb7, 0xbd, 0xd4, 0x91, 0xd6, 0x30, 0x52, 0xe4, 0x10, 0x16, 0x9d, 0xf5,
		0x2c, 0x5b, 0x58, 0x5b, 0x0f, 0x5b, 0x8a, 0xd4, 0x0a, 0xd4, 0x1f, 0x62,
		0x64, 0x0e, 0x15, 0x5c, 0xdd, 0x20, 0xfe, 0xb2, 0x22, 0x83, 0x2f, 0x52,
		0xff, 0x9c, 0xb4, 0x83, 0x94, 0x48, 0x9b, 0x18, 0xe9, 0xe3, 0xd3, 0xc9,
		0x4c, 0xbc, 0xa5, 0x14, 0x58, 0xbb, 0x8c, 0x72, 0x85, 0xf4, 0xdd, 0x89,
		0xa2, 0x6d, 0x21, 0xf5, 0xf3, 0x15, 0xda, 0xc6, 0x84, 0x2f, 0xe8, 0xe1,
		0x8f, 0xd7, 0x18, 0x4d, 0x27, 0x62, 0x4f, 0x1e, 0x7b, 0x49, 0xb3, 0x28,
		0xe7, 0x9c, 0x13, 0x60, 0x2e, 0xef, 0x6c, 0x1d, 0x53, 0xb4, 0x3a, 0xe9,
		0x4a, 0xca, 0x05, 0xee, 0x3e, 0x35, 0xb4, 0x97, 0x9f, 0x9f, 0xa3, 0x3e,
		0xb6, 0xc8, 0x55, 0xd2, 0xe5, 0x94, 0x67, 0x4d, 0xdc, 0x72, 0xce, 0x5d,
		0x05, 0xee, 0x5d, 0x4d, 0x75, 0x6c, 0xf1, 0x3e, 0x23, 0x95, 0x18, 0x40,
		0x37, 0x3a, 0x7b, 0xf9, 0xfa, 0x51, 0x8e, 0x8d, 0x8d, 0x4b, 0xa4, 0xd5,
		0x54, 0x4a, 0x6c, 0x6b, 0xf6, 0x58, 0xa6, 0xba, 0xfe, 0x2e, 0x6a, 0x43,
		0x39, 0x05, 0xbc, 0x95, 0xca, 0x2c, 0x1e, 0xe9, 0x63, 0xeb, 0x2a, 0xaa,
		0xa3, 0x39, 0x31, 0x4f, 0x73, 0xbc, 0xc4, 0x37, 0xe8, 0x6a, 0xf6, 0x40,
		0xa6, 0xbc, 0x7e, 0x3e, 0x7c, 0x29, 0xa7, 0xef, 0xee, 0x26, 0x75, 0x50,
		0xbb, 0x90, 0x72, 0xde, 0x0f, 0xb1, 0x83, 0xd4, 0x9d, 0x2d, 0xe9, 0x16,
		0x34, 0x7b, 0x0c, 0xad, 0x62, 0xe5, 0x62, 0x8e, 0xe4, 0xb5, 0x85, 0x15,
		0x0c, 0xbf, 0x95, 0xf3, 0xfa, 0x7e, 0x52, 0x5f, 0xb6, 0x3a, 0xb9, 0xae,
		0xd9, 0xc9, 0xb7, 0x92, 0xb6, 0x5e, 0x7e, 0xf9, 0x24, 0x67, 0x52, 0xfc,
		0x32, 0xa7, 0xc8, 0xc7, 0x48, 0xcb, 0x18, 0xea, 0xe0, 0xf1, 0x66, 0x27,
		0xde, 0x72, 0xda, 0x59, 0xb7, 0x2a, 0x67, 0x52, 0xcc, 0x3b, 0x1e, 0xa6,
		0x5a, 0xe2, 0x63, 0xb1, 0x94, 0xfb, 0x4f, 0xfa, 0x3a, 0xa9, 0x1e, 0x98,
		0xa0, 0xc8, 0xdb, 0x19, 0xed, 0xe6, 0x20, 0xe6, 0x36, 0x3b, 0xe1, 0x96,
		0xd5, 0xc3, 0xb6, 0xa7, 0x72, 0x26, 0xc5, 0x93, 0xc7, 0x3e, 0xd2, 0xec,
		0xec, 0xa6, 0xe4, 0x9a, 0x33, 0xfd, 0xae, 0x19, 0xff, 0x43, 0xbe, 0x2d,
		0x2b, 0xb1, 0x7f, 0x88, 0xe1, 0x46, 0xef, 0x3f, 0x48, 0xad, 0xca, 0xf3,
		0xd8, 0x75, 0xa6, 0xdf, 0x75, 0x2e, 0x17, 0xba, 0xb3, 0xc0, 0x0d, 0xed,
		0xcc, 0x6c, 0x14, 0x70, 0x3b, 0x9d, 0x25, 0xae, 0x9f, 0xc4, 0x9c, 0xa6,
		0x9d, 0xb6, 0x3e, 0xde, 0xbe, 0x8d, 0x5a, 0x7d, 0x9c, 0xfe, 0x7c, 0x8c,
		0x34, 0x2f, 0x6b, 0x1d, 0x8d, 0x36, 0x9c, 0xc2, 0x78, 0x7a, 0x78, 0xf6,
		0x2a, 0xaa, 0x95, 0xd3, 0x58, 0x71, 0x6c, 0xa2, 0xde, 0xc7, 0xf6, 0x66,
		0xe7, 0xdc, 0x72, 0xda, 0x59, 0x73, 0x01, 0xd5, 0x83, 0xa7, 0x51, 0xe4,
		0x44, 0x2a, 0x67, 0x13, 0x62, 0x05, 0x97, 0x36, 0x3b, 0xf7, 0x56, 0xb2,
		0xb2, 0x97, 0xca, 0x9e, 0x06, 0x45, 0x1d, 0x6e, 0xf0, 0xfa, 0x93, 0x1c,
		0x9f, 0xcd, 0xeb, 0xcd, 0x4e, 0xbe, 0x55, 0x5c, 0x52, 0x64, 0x68, 0x67,
		0x83, 0x62, 0x7e, 0x41, 0xba, 0x91, 0x91, 0xbc, 0x8d, 0xa4, 0x81, 0x6c,
		0x8f, 0xa3, 0x26, 0xf6, 0x38, 0x26, 0xd4, 0xd1, 0xcf, 0x9e, 0x17, 0x72,
		0xf6, 0x96, 0x13, 0xe9, 0x28, 0x69, 0x31, 0x47, 0x3a, 0x38, 0xf4, 0x6a,
		0xe3, 0xbb, 0xc3, 0xa3, 0x25, 0x36, 0x37, 0x7b, 0x20, 0x53, 0xda, 0x1c,
		0x5e, 0xbe, 0x99, 0xe1, 0xbc, 0xab, 0x35, 0x91, 0x36, 0x70, 0x74, 0x36,
		0x1f, 0x61, 0x59, 0x6f, 0x83, 0xfe, 0xfd, 0x1b, 0xa9, 0x2b, 0xeb, 0xd5,
		0xf3, 0x9a, 0x3d, 0x9e, 0xa9, 0xea, 0xa6, 0xf3, 0xa9, 0x1c, 0x6a, 0x50,
		0xe4, 0x5d, 0xa4, 0x22, 0x83, 0x38, 0x0f, 0x66, 0xb3, 0xf9, 0x4e, 0x6a,
		0x79, 0xb1, 0xb7, 0x50, 0x15, 0x4f, 0x55, 0x72, 0xb5, 0x97, 0xd8, 0xff,
		0x41, 0x83, 0x22, 0xd7, 0x48, 0x8b, 0xb2, 0x75, 0xf2, 0x1d, 0xa7, 0x9c,
		0x53, 0x2c, 0x71, 0xe0, 0xbd, 0x31, 0xb1, 0x3b, 0xb3, 0x1f, 0x64, 0x48,
		0xf4, 0xe9, 0x7f, 0xeb, 0x60, 0xc3, 0x4a, 0xca, 0x8d, 0x96, 0x6e, 0xeb,
		0xb3, 0x96, 0xf1, 0x7e, 0xce, 0xa9, 0xcb, 0xfb, 0xa8, 0xfe, 0x48, 0x1a,
		0x22, 0x6d, 0x24, 0x75, 0x64, 0x7f, 0x35, 0xb8, 0x67, 0x92, 0x87, 0xd0,
		0x12, 0x16, 0x76, 0x53, 0xde, 0x3b, 0xce, 0x2a, 0xa3, 0xc8, 0x61, 0xcc,
		0xcf, 0x3b, 0xb9, 0x87, 0xc7, 0x96, 0x50, 0x1e, 0x20, 0xbd, 0x9b, 0xad,
		0x3a, 0x0e, 0xa3, 0x34, 0xb9, 0x43, 0x68, 0x01, 0x33, 0x78, 0xe2, 0x01,
		0x8e, 0xe7, 0x15, 0xb9, 0x4e, 0x5a, 0xca, 0x91, 0x02, 0x6b, 0xc7, 0xf9,
		0x88, 0xb6, 0x22, 0xcf, 0xb4, 0x67, 0x57, 0xfd, 0x3e, 0x2c, 0x9f, 0xac,
		0xdc, 0x5b, 0xcd, 0x8a, 0x8b, 0x1a, 0x3c, 0xae, 0x7a, 0x85, 0xd1, 0x5e,
		0x76, 0x3b, 0xbd, 0x8d, 0xfc, 0x86, 0x9b, 0x4e, 0x21, 0xb3, 0xe6, 0xae,
		0x9c, 0xfe, 0xfc, 0xfb, 0xdf, 0xb7, 0xd4, 0x4b, 0x27, 0x33, 0x99, 0xe9,
		0xbc, 0x4d, 0x7a, 0xe0, 0xfb, 0x6c, 0x02, 0xfb, 0x87, 0x75, 0xd4, 0x46,
		0xd9, 0x22, 0xfb, 0x67, 0x52, 0xf8, 0x1f, 0x14, 0x4a, 0xfc, 0xf4, 0xc6,
		0x29, 0x57, 0xf3, 0x9b, 0xd9, 0x04, 0x38, 0x80, 0x9e, 0x66, 0x27, 0x37,
		0xdd, 0x2c, 0x29, 0x32, 0x78, 0x05, 0x83, 0x73, 0x29, 0xf7, 0xf2, 0x03,
		0xae, 0x6d, 0x76, 0x52, 0xd3, 0x55, 0xa7, 0x6c, 0xc5, 0xb0, 0x50, 0x3c,
		0xc5, 0x9e, 0xfe, 0xfe, 0x02, 0xac, 0xdd, 0x70, 0x07, 0xf8, 0xa7, 0xc1,
		0x3c, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60,
		0x82,
	}),
)
