import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: 'shrtn - link infrastructure',
  description: 'A URL shortener built for real throughput.',
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      className={`h-full antialiased`}
    >
      <body className="min-h-full flex flex-col">{children}</body>
    </html>
  );
}
