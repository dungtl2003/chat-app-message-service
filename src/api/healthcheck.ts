import {Service, ServiceStatus} from "@/loaders/services/service";
import express, {Request, Response} from "express";

type HealthStatus = "UP" | "DOWN";

interface HealthResponse {
    status: HealthStatus;
}

function checkHealth(services?: Service[]) {
    return function (_: Request, response: Response<HealthResponse>) {
        if (!services) {
            response.status(200).send({
                status: "UP",
            });
            return;
        }

        for (const service of services!) {
            if (service.getStatus() !== ServiceStatus.INITIALIZED) {
                response.status(200).send({
                    status: "DOWN",
                });
                return;
            }
        }

        response.status(200).send({
            status: "UP",
        });
    };
}

export default (services?: Service[]) => {
    const router = express.Router();

    router.get("/", checkHealth(services));

    return router;
};
