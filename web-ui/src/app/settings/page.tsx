import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

export default function SettingsPage() {
  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Settings</h1>
        <p className="text-muted-foreground">
          Configure platform settings and preferences
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Platform Configuration</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">
            Platform settings and configuration options will be implemented here.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}