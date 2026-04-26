<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('habits', function (Blueprint $table) {
            $table->id();
            $table->foreignId('sprint_participant_id')->nullable()->constrained('sprint_participants')->nullOnDelete();
            $table->string('name', 80);
            $table->unsignedInteger('display_order')->default(0);
            $table->timestamps();
            $table->softDeletes();

            $table->index('display_order');
            $table->index('deleted_at');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('habits');
    }
};
