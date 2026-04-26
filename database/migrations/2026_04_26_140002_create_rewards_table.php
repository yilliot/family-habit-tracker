<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('rewards', function (Blueprint $table) {
            $table->id();
            $table->foreignId('person_id')->constrained('people')->cascadeOnDelete();
            $table->string('name', 80);
            $table->unsignedInteger('cost');
            $table->timestamp('achieved_at')->nullable();
            $table->timestamps();

            $table->index(['person_id', 'achieved_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('rewards');
    }
};
