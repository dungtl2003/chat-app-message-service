import {execa} from "execa";

async function findContainerId(node: string, opt: string): Promise<string> {
    const cmd = `docker ps \
      --filter "status=running" \
      --filter "label=custom.project=chat" \
      --filter "label=custom.service=${node}" \
      --filter "label=custom.option=${opt}" \
      --no-trunc \
      -q`;
    const {stdout: containerId} = await execa({shell: true})`${cmd}`;
    console.log(`${node}: ${containerId}`);
    return containerId;
}

async function waitForIdGenerator(containerId: string, opt: string) {
    // TODO: update this to be more real
    await execa({shell: true})`sleep 5`;
    const snowflake = await findContainerId("id-generator", opt);
    if (snowflake !== "") {
        console.log(`ID generator container ${containerId} is running`);
    } else {
        console.log(`ID generator container ${containerId} is not running`);
        throw new Error();
    }
}

async function main() {
    console.log("\nFinding container ids...");
    const snowflakeTls = await findContainerId("id-generator", "tls");
    const snowflakeNonTls = await findContainerId("id-generator", "non-tls");

    console.log("\nWaiting for nodes...");
    await Promise.all([
        waitForIdGenerator(snowflakeTls, "tls"),
        waitForIdGenerator(snowflakeNonTls, "non-tls"),
    ]);

    console.log("\nAll nodes up:");
    const cmd = `docker compose -f ${process.env.COMPOSE_FILE} ps`;
    const {stdout} = await execa({shell: true})`${cmd}`;
    console.log(stdout);
}

main();
