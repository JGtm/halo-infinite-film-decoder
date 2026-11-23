//go:build windows

package main

import (
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// processeur lit le nom du processeur dans le registre.
func processeur() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\CentralProcessor\0`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer func() { _ = k.Close() }()
	nom, _, err := k.GetStringValue("ProcessorNameString")
	if err != nil {
		return ""
	}
	return strings.Join(strings.Fields(nom), " ")
}

// etatMemoire est la structure MEMORYSTATUSEX de l'API Windows.
type etatMemoire struct {
	Taille, Charge                                                                             uint32
	PhysTotale, PhysLibre, FichierTotal, FichierLibre, VirtTotale, VirtLibre, VirtEtendueLibre uint64
}

// memoireMio rend la memoire physique totale, en Mio (0 si indisponible).
func memoireMio() uint64 {
	dll := syscall.NewLazyDLL("kernel32.dll")
	proc := dll.NewProc("GlobalMemoryStatusEx")
	var e etatMemoire
	e.Taille = uint32(unsafe.Sizeof(e))
	r, _, _ := proc.Call(uintptr(unsafe.Pointer(&e)))
	if r == 0 {
		return 0
	}
	return e.PhysTotale >> 20
}
