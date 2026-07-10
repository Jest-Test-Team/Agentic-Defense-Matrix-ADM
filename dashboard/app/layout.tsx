import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "ADM Battle Console",
  description: "Live status of the Agentic Defense Matrix red/blue/green exercise",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
