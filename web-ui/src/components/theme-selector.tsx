"use client"

import { useState } from 'react'
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet"
import { Badge } from "@/components/ui/badge"
import { Palette, Check, Sparkles } from "lucide-react"
import { useCustomTheme } from "@/contexts/theme-context"

export function ThemeSelector() {
  const { currentTheme, setTheme, themes } = useCustomTheme()
  const [isOpen, setIsOpen] = useState(false)

  const handleThemeChange = (themeId: string) => {
    setTheme(themeId)
    setIsOpen(false)
  }

  return (
    <Sheet open={isOpen} onOpenChange={setIsOpen}>
      <SheetTrigger asChild>
        <Button
          variant="ghost"
          size="sm"
          className="gap-2 text-green-100 hover:text-white hover:bg-green-700"
        >
          <Palette className="w-4 h-4" />
          <span className="hidden sm:inline">Themes</span>
        </Button>
      </SheetTrigger>
      <SheetContent className="w-[400px] sm:w-[540px]">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            <Sparkles className="w-5 h-5" />
            Choose Your Theme
          </SheetTitle>
          <SheetDescription>
            Select from our collection of beautiful themes to personalize your dashboard experience.
          </SheetDescription>
        </SheetHeader>

        <div className="mt-6 space-y-4">
          <div className="flex items-center gap-2 mb-4">
            <Badge variant="outline" className="bg-blue-50 text-blue-700 border-blue-200">
              Current: {currentTheme.name}
            </Badge>
          </div>

          <div className="grid gap-4">
            {themes.map((theme) => (
              <Card
                key={theme.id}
                className={`cursor-pointer transition-all hover:shadow-lg ${
                  currentTheme.id === theme.id
                    ? 'ring-2 ring-blue-500 shadow-lg'
                    : 'hover:shadow-md'
                }`}
                onClick={() => handleThemeChange(theme.id)}
              >
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="text-base flex items-center gap-2">
                        {theme.name}
                        {currentTheme.id === theme.id && (
                          <Check className="w-4 h-4 text-green-600" />
                        )}
                      </CardTitle>
                      <p className="text-sm text-muted-foreground mt-1">
                        {theme.description}
                      </p>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="pt-0">
                  {/* Theme Preview */}
                  <div className="space-y-3">
                    {/* Background Preview */}
                    <div className={`h-8 rounded-lg bg-gradient-to-r from-${theme.colors.background.from} via-${theme.colors.background.via} to-${theme.colors.background.to} border`}>
                      <div className="h-full w-full rounded-lg bg-gradient-to-r opacity-80" />
                    </div>

                    {/* Cards Preview */}
                    <div className="grid grid-cols-4 gap-2">
                      <div className={`h-6 rounded bg-gradient-to-br from-${theme.colors.cards.applications.from} to-${theme.colors.cards.applications.to}`} />
                      <div className={`h-6 rounded bg-gradient-to-br from-${theme.colors.cards.workflows.from} to-${theme.colors.cards.workflows.to}`} />
                      <div className={`h-6 rounded bg-gradient-to-br from-${theme.colors.cards.resources.from} to-${theme.colors.cards.resources.to}`} />
                      <div className={`h-6 rounded bg-gradient-to-br from-${theme.colors.cards.users.from} to-${theme.colors.cards.users.to}`} />
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

          <div className="pt-4 border-t">
            <p className="text-xs text-muted-foreground text-center">
              Your theme preference is automatically saved and will persist across sessions.
            </p>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}