<?php
// PSR-4 autoloader for the Nmid SDK. Require this file once, then use any
// Nmid\* class directly.

spl_autoload_register(function (string $class): void {
    $prefix = 'Nmid\\';
    $baseDir = __DIR__ . '/src/';

    if (strncmp($class, $prefix, strlen($prefix)) !== 0) {
        return;
    }

    $relative = substr($class, strlen($prefix));
    $file = $baseDir . str_replace('\\', '/', $relative) . '.php';
    if (is_file($file)) {
        require $file;
    }
});
