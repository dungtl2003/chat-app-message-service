enum ServiceStatus {
    INITIALIZE_FAILED,
    INITIALIZING,
    INITIALIZED,
}

interface Service {
    getStatus(): ServiceStatus;
}

export {ServiceStatus, Service};
