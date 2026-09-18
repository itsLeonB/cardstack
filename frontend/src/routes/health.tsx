import { createFileRoute } from "@tanstack/react-router"
import { HealthCheck } from "@/components/health-check"

export const Route = createFileRoute("/health")({ component: HealthCheck })
