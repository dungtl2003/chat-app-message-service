/**
 * Returns a random number between min (inclusive) and max (exclusive)
 */
function getRandomArbitrary(min: number, max: number): number {
    return Math.random() * (max - min) + min;
}

/**
 * Returns a random integer between min (inclusive) and max (inclusive).
 * The value is no lower than min (or the next integer greater than min
 * if min isn't an integer) and no greater than max (or the next integer
 * lower than max if max isn't an integer).
 * Using Math.round() will give you a non-uniform distribution!
 */
function getRandomInt(min: number, max: number): number {
    min = Math.ceil(min);
    max = Math.floor(max);
    return Math.floor(Math.random() * (max - min + 1)) + min;
}

function objectToString(o: {[s: string]: string[] | undefined}): string {
    return (
        "{" +
        Object.entries(o)
            .map(([key, value]) => `${key}: ${value}`)
            .join(", ") +
        "}"
    );
}

function containsOnlyDigits(str: string): boolean {
    return /^\d+$/.test(str);
}

export {getRandomArbitrary, getRandomInt, objectToString, containsOnlyDigits};
